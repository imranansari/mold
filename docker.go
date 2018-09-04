package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

// ContainerConfig holds all configs needed to run the docker container
type ContainerConfig struct {
	Name      string // name of the running container
	shortName string
	Container *container.Config
	Host      *container.HostConfig
	Network   *network.NetworkingConfig

	id    string
	state *types.ContainerState
}

// ID returns the id as set by docker. This gets set once the container has
// started
func (cc *ContainerConfig) ID() string {
	return cc.id
}

// IsRunning reports if the container is running per the local state.
func (cc *ContainerConfig) IsRunning() bool {
	return cc.state.Running
}

// DefaultContainerConfig contains just the image name
func DefaultContainerConfig(imageName string) *ContainerConfig {
	return &ContainerConfig{
		Container: &container.Config{Image: imageName, Volumes: map[string]struct{}{}},
		Host:      &container.HostConfig{Binds: []string{}, Mounts: []mount.Mount{}},
		Network:   &network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{}},
	}
}

// Docker provides a wrapper to perform rudamentary docker operations
type Docker struct {
	cli *client.Client
}

// NewDocker returns a new docker client helper using the given uri.  If uri is
// not provided it uses the default env. client
func NewDocker(uri string) (*Docker, error) {
	var (
		hcli    *http.Client
		version = os.Getenv("DOCKER_API_VERSION")
	)

	if len(uri) == 0 {
		if runtime.GOOS == "windows" {
			uri = windowsDockerURI
			dockerSockFile = windowsDockerURI
		} else {
			uri = linuxDockerURI
		}
	}

	if runtime.GOOS == "windows" {
		os.Setenv("HOME", "C:/Users/"+os.Getenv("USERNAME"))
	}

	//if version == "" {
	//      client.
	//}
	cli, err := client.NewClient(uri, version, hcli, nil)
	if err != nil {
		return nil, err
	}
	return &Docker{cli: cli}, nil
}

// Client returns the raw docker client
func (dkr *Docker) Client() *client.Client {
	return dkr.cli
}

// ImageAvailableLocally returns true if the image is locally available
func (dkr *Docker) ImageAvailableLocally(imageName string) bool {
	if _, _, err := dkr.cli.ImageInspectWithRaw(context.Background(), imageName); err == nil {
		return true
	}
	return false
}

// StartContainer creates and starts a container with the given config updating
// the state of the ContainerConfig.  It also pulls the base image if not locally
// available. This is a non-blocking call.
func (dkr *Docker) StartContainer(cc *ContainerConfig, wr *Log, prefix string) error {
	if !dkr.ImageAvailableLocally(cc.Container.Image) {
		if err := dkr.PullImage(cc.Container.Image, nil, wr, prefix); err != nil {
			return err
		}
	}

	c, err := dkr.cli.ContainerCreate(context.Background(), cc.Container, cc.Host, cc.Network, cc.Name)
	if err != nil {
		return err
	}
	cc.id = c.ID

	if err = dkr.cli.ContainerStart(context.Background(), cc.id, types.ContainerStartOptions{}); err == nil {

		var cont types.ContainerJSON
		if cont, err = dkr.cli.ContainerInspect(context.Background(), cc.id); err == nil {
			cc.state = cont.State
			if len(cc.Name) == 0 {
				cc.Name = cont.Name[1:]
			}
		}
	}
	return err
}

// StopContainer stops a container
func (dkr *Docker) StopContainer(containerID string, timeout time.Duration) error {
	return dkr.cli.ContainerStop(context.Background(), containerID, &timeout)
}

// TailLogs tail container logs to the given writer
// Attach stdout and stderr of the container to stdout
func (dkr *Docker) TailLogs(containerID string, wr io.Writer, prefix string) error {
	opts := types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Follow:     true,
		Timestamps: true,
	}
	r, err := dkr.cli.ContainerLogs(context.Background(), containerID, opts)
	if err != nil {
		return err
	}
	defer r.Close()
	buf := bufio.NewReader(r)

	for {
		var b []byte
		if b, err = buf.ReadBytes('\n'); err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
		// First 8 bytes are the stream type
		// Append the specified prefix
		wr.Write([]byte(prefix + " "))
		// Remove the date from the timestamp
		if len(b) > 39 {
			wr.Write(append(append(b[19:32], ' '), b[39:]...))
		} else {
			wr.Write(b)
		}
	}
	return err
}

// RemoveContainer removes a container
func (dkr *Docker) RemoveContainer(containerID string, force bool) error {
	options := types.ContainerRemoveOptions{Force: force}
	return dkr.cli.ContainerRemove(context.Background(), containerID, options)
}

// BuildImageOfContainer creates an image of a container
func (dkr *Docker) BuildImageOfContainer(containerID string, reference string) error {
	options := types.ContainerCommitOptions{
		Reference: reference,
	}
	_, err := dkr.cli.ContainerCommit(context.Background(), containerID, options)
	return err
}

// BuildImageAsync builds a docker images based on the config and writes the log out
// to the the specified Writer.  This is an async call.
func (dkr *Docker) BuildImageAsync(ic *ImageConfig, logWriter io.Writer, prefix string, done chan bool) error {
	bldCxt, err := tarDirectory(ic.Context)
	if err != nil {
		return err
	}
	opts := types.ImageBuildOptions{
		Dockerfile: ic.Dockerfile,
		Tags:       ic.DefaultRegistryPaths(),
		Remove:     true, // remove intermediate images
		NoCache:    !ic.CachedBuild,
	}

	rps := ic.CustomRegistryPaths()
	opts.Tags = append(opts.Tags, rps...)

	rsp, err := dkr.cli.ImageBuild(context.Background(), bldCxt, opts)
	if err != nil {
		return err
	}

	go func() {
		logWriter.Write([]byte(fmt.Sprintf("%s Starting a routine to watch the image build...\n", prefix)))
		var (
			buf = bufio.NewReader(rsp.Body)
			e   error
		)
		for {
			b, err := buf.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					e = err
				}
				break
			}
			b = b[:len(b)-1]
			var m map[string]interface{}
			if err = json.Unmarshal(b, &m); err != nil {
				logWriter.Write([]byte(fmt.Sprintf("%s ERR %s %s\n", prefix, err, b)))
				e = err
				break
			}

			if v, ok := m["error"]; ok {
				e = fmt.Errorf(v.(string))
				break
			}

			if v, ok := m["stream"]; ok {
				str := v.(string)
				logWriter.Write([]byte(prefix + " " + str))
			} else {
				d := append([]byte(prefix+" "), b...)
				logWriter.Write(d)
			}
		}
		rsp.Body.Close()
		done <- (e == nil)
	}()
	return nil
}

// RemoveImage locally from the host
func (dkr *Docker) RemoveImage(imageID string, force bool, cleanUp bool) error {
	options := types.ImageRemoveOptions{Force: force}
	if cleanUp {
		options.PruneChildren = true
	}
	_, err := dkr.cli.ImageRemove(context.Background(), imageID, options)
	return err
}

// CreateNetwork creates a bridge network
func (dkr *Docker) CreateNetwork(name string) (string, error) {
	opts := types.NetworkCreate{Driver: "bridge", CheckDuplicate: false}
	rsp, err := dkr.cli.NetworkCreate(context.Background(), name, opts)
	if err != nil {
		return "", err
	}
	return rsp.ID, nil
}

// RemoveNetwork removes the network from the host
func (dkr *Docker) RemoveNetwork(networkID string) error {
	return dkr.cli.NetworkRemove(context.Background(), networkID)
}

func (dkr *Docker) GetAuthBase64(authConfig types.AuthConfig) (string, error) {
	auth, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(auth), nil
}

func (dkr *Docker) GetImageList() ([]types.ImageSummary, error) {
	return dkr.cli.ImageList(context.Background(), types.ImageListOptions{All: true})
}

// PushImage pushes a local docker image up to a registry
func (dkr *Docker) PushImage(imageRef string, authCfg *types.AuthConfig, logWriter io.Writer, prefix string) error {
	opts := types.ImagePushOptions{}
	if authCfg != nil {
		if a, err := dkr.GetAuthBase64(*authCfg); err != nil {
			logWriter.Write([]byte(fmt.Sprintf("%s Error: %s\n", prefix, err.Error())))
			return err
		} else {
			opts.RegistryAuth = a
		}
	}

	logWriter.Write([]byte(prefix + " Publishing image: " + imageRef + "\n"))
	rsp, err := dkr.cli.ImagePush(context.Background(), imageRef, opts)
	if err != nil {
		return err
	}
	defer rsp.Close()
	buf := bufio.NewReader(rsp)

	for {
		var b []byte
		if b, err = buf.ReadBytes('\n'); err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}

		var m map[string]interface{}
		if err = json.Unmarshal(b, &m); err != nil {
			break
		}

		if st, ok := m["status"]; ok {
			status := st.(string)
			logWriter.Write([]byte(prefix + " " + status + "\n"))
		} else if er, ok := m["error"]; ok {
			errStr := er.(string)
			logWriter.Write([]byte(prefix + " " + errStr + "\n"))
			err = fmt.Errorf(errStr)
			break
		} else {
			logWriter.Write(append([]byte(prefix), b...))
		}
	}

	return err
}

// PullImage pulls a remote image from a registry down locally
func (dkr *Docker) PullImage(imageRef string, authCfg *types.AuthConfig, logWriter io.Writer, prefix string) error {
	opts := types.ImagePullOptions{}
	if authCfg != nil {
		if a, err := dkr.GetAuthBase64(*authCfg); err != nil {
			logWriter.Write([]byte(fmt.Sprintf("%s Error: %s\n", prefix, err.Error())))
			return err
		} else {
			opts.RegistryAuth = a
		}
	}
	rsp, err := dkr.cli.ImagePull(context.Background(), imageRef, opts)
	if err != nil {
		return err
	}
	defer rsp.Close()
	buf := bufio.NewReader(rsp)

	logWriter.Write([]byte(fmt.Sprintf("%s Pulling image: %s\n", prefix, imageRef)))
	for {
		var b []byte
		if b, err = buf.ReadBytes('\n'); err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
		b = b[:len(b)-1]

		var m imgPullProgress
		if err = json.Unmarshal(b, &m); err != nil {
			break
		}

		if m.ProgressDetail.Current == m.ProgressDetail.Total && m.ProgressDetail.Total > 0 {
			logWriter.Write([]byte(fmt.Sprintf("%s %s: %s %d bytes\n", prefix, m.Status, m.ID, m.ProgressDetail.Total)))
		}

	}
	if err == nil {
		logWriter.Write([]byte(fmt.Sprintf("%s Pulled image: %s\n", prefix, imageRef)))
	}

	return err
}

type imgPullProgressDetail struct {
	Current  int
	Total    int
	Progress string
}

func (ipd *imgPullProgressDetail) Percent() float64 {
	return float64(ipd.Current) / float64(ipd.Total) * 100
}

type imgPullProgress struct {
	Status         string
	ID             string
	ProgressDetail imgPullProgressDetail
}

// DockerAuthConfig contains the auth config to perform registry operations
type DockerAuthConfig struct {
	Auths       map[string]types.AuthConfig `json:"auths"`
	HTTPHeaders map[string]string           `json:"HttpHeaders"`
}

// DockerHubAuth returns the authconfig for docker hub
func (dac *DockerAuthConfig) DockerHubAuth() *types.AuthConfig {
	if dac.Auths == nil {
		return nil
	}
	for k, v := range dac.Auths {
		pp := strings.Split(k, ".")
		// make sure the fqdn has atleast 3 parts for docker hub registry
		if len(pp) > 2 {
			if pp[len(pp)-2] == "docker" {
				if v.ServerAddress == "" {
					v.ServerAddress = k
				}
				return &v
			}
		}
	}
	return nil
}

func readDockerAuthConfig(cfgfile string) (*DockerAuthConfig, error) {
	cfile := cfgfile
	if cfile == "" {
		home := os.Getenv("HOME")
		cfile = filepath.Join(home, ".docker/config.json")
	}
	b, err := ioutil.ReadFile(cfile)
	if err == nil {
		var dac DockerAuthConfig
		err = json.Unmarshal(b, &dac)
		return &dac, err
	}
	return nil, err
}
