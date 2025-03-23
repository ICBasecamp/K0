package main

import (
	"context"
	// "fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

type DockerClient struct {
	cli *client.Client
	ctx context.Context
}

func CreateDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerClient{
		cli: cli,
		ctx: context.Background(),
	}, nil
}

func (dc *DockerClient) StartContainer(imageName string) (string, error) {
	out, err := dc.cli.ImagePull(dc.ctx, imageName, image.PullOptions{})
	if err != nil {
		panic(err)
	}
	defer out.Close()

	resp, err := dc.cli.ContainerCreate(dc.ctx, &container.Config{
		Image: imageName,
	}, nil, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := dc.cli.ContainerStart(dc.ctx, resp.ID, container.StartOptions{}); err != nil {
		panic(err)
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}

	logs, err := dc.cli.ContainerLogs(dc.ctx, resp.ID, options)
	if err != nil {
		panic(err)
	}
	defer logs.Close()

	io.Copy(os.Stdout, logs)

	return resp.ID, nil
}

// example usage
func main() {
	dc, err := CreateDockerClient()
	if err != nil {
		panic(err)
	}

	dc.StartContainer("hello-world")
}