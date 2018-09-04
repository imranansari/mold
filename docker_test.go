package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/docker/docker/api/types"
)

func Test_Docker(t *testing.T) {
	d, err := NewDocker("wrong address")
	if err == nil {
		t.Fatal("should be uri error")
	}

	d, err = NewDocker("")
	if err != nil {
		t.Fatal(err)
	}
	if d.Client() == nil {
		t.Fatal("failed to init docker client")
	}
	d = nil

	if d, err = NewDocker("unix:///var/run/docker.sock"); err != nil {
		t.Fatal(err)
	}

	if d.Client() == nil {
		t.Fatal("failed to init docker client")
	}
}
func Test_Docker_StartContainer(t *testing.T) {
	d, err := NewDocker("")
	if err != nil {
		t.Fatal(err)
	}

	cc := DefaultContainerConfig("testsdkxjkchvsldkjf")
	err = d.StartContainer(cc, nil, "prefix")
	if err == nil {
		t.Fatal("should be pull error")
	}

	_, err = d.cli.ContainerCreate(context.Background(), cc.Container, cc.Host, cc.Network, cc.Name)
	if err == nil {
		t.Fatal("should be container create error")
	}
}

func Test_Docker_PullImage(t *testing.T) {
	d, _ := NewDocker("")
	if err := d.PullImage("busybox:latest", nil, os.Stdout, ""); err != nil {
		t.Fatal(err)
	}

	d.RemoveImage("busybox:latest", true, false)

	if err := d.PullImage("nosuchrepo:latest", nil, os.Stdout, ""); err == nil {
		t.Fatal("should be reported when failing to pull an image")
	}
}

func Test_ImageAvailableLocally(t *testing.T) {
	d, _ := NewDocker("")
	if exists := d.ImageAvailableLocally("secretImage"); exists == true {
		t.Fatal("should return false")
	}
}

func TestNewDocker(t *testing.T) {
	type args struct {
		uri string
	}

	tests := []struct {
		name    string
		args    args
		want    *Docker
		wantErr bool
	}{{
		name:    "empty",
		args:    args{""},
		wantErr: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDocker(tt.args.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDocker() error = %s, wantErr %t\n", err.Error(), tt.wantErr)
				return
			}
		})
	}
}

func Test_Docker_BuildImageOfContainer(t *testing.T) {
	d, _ := NewDocker("unix:///var/run/docker.sock")
	if err := d.BuildImageOfContainer("", ""); err == nil {
		t.Fatal("expected error")
	}
}

func Test_GetAuthBase64(t *testing.T) {
	d, _ := NewDocker("unix:///var/run/docker.sock")
	if _, err := d.GetAuthBase64(types.AuthConfig{Username: "k"}); err != nil {
		t.Fatal(err.Error())
	}
}

func Test_PushImage(t *testing.T) {
	d, _ := NewDocker("unix:///var/run/docker.sock")
	a := &types.AuthConfig{}
	if err := d.PushImage("rp", a, os.Stdout, fmt.Sprintf("[publish/%s]", "rp")); err == nil {
		t.Fatal("should be auth error")
	}

	authConf := &types.AuthConfig{Username: "Unknown"}
	if err := d.PushImage("rp", authConf, os.Stdout, fmt.Sprintf("[publish/%s]", "rp")); err == nil {
		t.Fatal("should be push error")
	}
}

func Test_DockerHubAuth(t *testing.T) {
	aC := DockerAuthConfig{
		Auths: make(map[string]types.AuthConfig, 0),
	}
	aC.Auths["docker"] = types.AuthConfig{Username: "UnknownEmpty"}

	a := aC.DockerHubAuth()
	if a != nil {
		t.Error("should be nil docker hub auth config")
	}

	aC.Auths["docker.docker.docker"] = types.AuthConfig{Username: "Unknown"}

	a = aC.DockerHubAuth()
	if a == nil {
		t.Error("docker hub auth config should be not nil")
	}
}
