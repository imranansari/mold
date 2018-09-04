package main

import (
	"testing"
	"time"
	"github.com/docker/docker/api/types"
	"io"
	"github.com/docker/docker/client"
	"errors"
	"github.com/docker/docker/api/types/network"
)

func Test_buildServiceStates_AddsFirstEightCharsOfGitHashToContainerName(t *testing.T) {
	moldConfig := &MoldConfig{
		Services: []DockerRunConfig{
			{
				Name: "servicename",
			},
		},
	}

	result, err := buildServiceStates(moldConfig, "irrelevant", func() string {
		return "aprettylonghash"
	})
	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}

	expected := "servicename-aprettylonghash"
	if result[0].Name != expected {
		t.Fatalf("expected %v but got %v", expected, result[0].Name)
	}
}

func Test_buildServiceStates_SetsNetworkAliasToServiceNameWithoutHash(t *testing.T) {
	moldConfig := &MoldConfig{
		Services: []DockerRunConfig{
			{
				Name: "servicename",
			},
		},
		RepoName: "repo",
		BranchTag: "branch",
	}

	result, err := buildServiceStates(moldConfig, "foo", func() string {
		return "thehashk"
	})
	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}

	networkConfigName := "repo-branch"
	var endpointConfig *network.EndpointSettings
	var ok bool
	if endpointConfig, ok = result[0].Network.EndpointsConfig[networkConfigName]; !ok {
		t.Fatalf("expected to find a network config with name %v", networkConfigName)
	}

	if !contains(endpointConfig.Aliases, "servicename") {
		t.Fatalf("Expected a network alias called servicename but didn't find one")
	}
}

// why, golang? why?
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

type remove func(containerID string, force bool) error

type mockDocker struct {
	removeFunc remove
}

func (mockDocker) BuildImageOfContainer(containerID string, reference string) error {
	panic("implement me")
}

func (mockDocker) BuildImageAsync(ic *ImageConfig, logWriter io.Writer, prefix string, done chan bool) error {
	panic("implement me")
}

func (mockDocker) Client() *client.Client {
	panic("implement me")
}

func (mockDocker) CreateNetwork(name string) (string, error) {
	panic("implement me")
}

func (mockDocker) GetImageList() ([]types.ImageSummary, error) {
	panic("implement me")
}

func (mockDocker) ImageAvailableLocally(imageName string) bool {
	panic("implement me")
}

func (md mockDocker) RemoveContainer(containerID string, force bool) error {
	return md.removeFunc(containerID, force)
}

func (mockDocker) RemoveImage(imageID string, force bool, cleanUp bool) error {
	panic("implement me")
}

func (mockDocker) RemoveNetwork(networkID string) error {
	panic("implement me")
}

func (mockDocker) PushImage(imageRef string, authCfg *types.AuthConfig, logWriter io.Writer, prefix string) error {
	panic("implement me")
}

func (mockDocker) StartContainer(cc *ContainerConfig, wr *Log, prefix string) error {
	panic("implement me")
}

func (mockDocker) StopContainer(containerID string, timeout time.Duration) error {
	panic("implement me")
}

func (mockDocker) TailLogs(containerID string, wr io.Writer, prefix string) error {
	panic("implement me")
}

func Test_removeContainers_RemovesAllByID(t *testing.T) {
	idsRemoved := make(map[string]bool)

	docker := mockDocker{
		removeFunc: func(containerID string, force bool) error {
			idsRemoved[containerID] = true
			return nil
		},
	}

	states := containerStates{
		&containerState{
			ContainerConfig: &ContainerConfig{
				id: "foo",
			},
		},
		&containerState{
			ContainerConfig: &ContainerConfig{
				id: "bar",
			},

		},
	}

	removeContainers(states, docker, nil)

	if !idsRemoved["foo"] {
		t.Error("Expected to have removed foo")
	}

	if !idsRemoved["bar"] {
		t.Error("Expected to have removed bar")
	}
}

func Test_removeContainers_AppendsErrors(t *testing.T) {
	docker := mockDocker{
		removeFunc: func(containerID string, force bool) error {
			return errors.New("bad")
		},
	}

	states := containerStates{
		&containerState{
			ContainerConfig: &ContainerConfig{
				id: "foo",
			},
		},
		&containerState{
			ContainerConfig: &ContainerConfig{
				id: "bar",
			},

		},
	}

	err := removeContainers(states, docker, nil)

	if err.Error() != "bad\nbad" {
		t.Errorf("Expected bad\\nbad but got %v", err)
	}
}