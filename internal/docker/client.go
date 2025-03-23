package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
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

func (dc *DockerClient) BuildAndStartContainer(imageName string, buildContextPath string) (TerminalResponse, error) {
	buildContext, err := archive.TarWithOptions(buildContextPath, &archive.TarOptions{})
	if err != nil {
		panic(err)
	}
	defer buildContext.Close()

	buildOptions := types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile",
		Remove:     true,
	}

	response, err := dc.cli.ImageBuild(dc.ctx, buildContext, buildOptions)
	if err != nil {
		panic(err)
	}

	// replace io.Discard with os.Stdout for debugging and logging
	_, err = io.Copy(io.Discard, response.Body)
	if err != nil {
		return TerminalResponse{}, fmt.Errorf("failed to read build output: %w", err)
	}

	defer response.Body.Close()

	startResponse, err := dc.StartContainer(imageName + ":latest", false)
	if err != nil {
		panic(err)
	}

	return startResponse, nil
}

func (dc *DockerClient) StartContainer(imageName string, pull bool) (TerminalResponse, error) {
	if pull {
		out, err := dc.cli.ImagePull(dc.ctx, imageName, image.PullOptions{})
		if err != nil {
			panic(err)
		}
		// Process the image pull output before closing
		io.Copy(io.Discard, out)
		defer out.Close()
	}

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
	return TerminalResponse{
		ID:     resp.ID,
		Result: logs,
	}, nil
}

// functions for debugging

func PrintTerminalResponse(response TerminalResponse) {
	io.Copy(os.Stdout, response.Result)
	defer response.Result.Close()
}

func (dc *DockerClient) ListImages() {

	images, err := dc.cli.ImageList(dc.ctx, image.ListOptions{})
	if err != nil {
		panic(err)
	}

	for _, image := range images {
		fmt.Println(image.ID)
	}
	
}


// example usage
// func main() {
// 	dc, err := CreateDockerClient()
// 	if err != nil {
// 		panic(err)
// 	}

// 	dc.StartContainer("hello-world")
// }
