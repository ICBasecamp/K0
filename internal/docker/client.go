package docker

import (
	"context"
	// "fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

type DockerClient struct {
	cli *client.Client
	ctx context.Context
}

type TerminalResponse struct {
	ID     string
	Result io.ReadCloser
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

func (dc *DockerClient) StartContainer(imageName string) (TerminalResponse, error) {
	out, err := dc.cli.ImagePull(dc.ctx, imageName, image.PullOptions{})
	if err != nil {
		panic(err)
	}
	// Process the image pull output before closing
	io.Copy(io.Discard, out)
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
	// Don't defer close here as we're returning the logs to be consumed elsewhere

	return TerminalResponse{
		ID:     resp.ID,
		Result: logs,
	}, nil
}

// example usage
// func main() {
// 	dc, err := CreateDockerClient()
// 	if err != nil {
// 		panic(err)
// 	}

// 	dc.StartContainer("hello-world")
// }
