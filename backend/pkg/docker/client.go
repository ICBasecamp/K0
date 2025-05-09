package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ICBasecamp/K0/backend/pkg/s3"
	"github.com/ICBasecamp/K0/internal/github"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

type DockerClient struct {
	cli      *client.Client
	ctx      context.Context
	s3Client *s3.S3Client
}

type TerminalResponse struct {
	ID     string
	Result io.ReadCloser
}

func CreateDockerClient(s3Client *s3.S3Client) (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerClient{
		cli:      cli,
		ctx:      context.Background(),
		s3Client: s3Client,
	}, nil
}

func (dc *DockerClient) BuildAndStartContainer(imageName string, s3Key string) (TerminalResponse, error) {
	// Download build context from S3
	buildContext, err := dc.s3Client.DownloadFromS3(s3Key)
	if err != nil {
		return TerminalResponse{}, fmt.Errorf("failed to download build context from S3: %w", err)
	}
	defer buildContext.Close()

	buildOptions := types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile",
		Remove:     true,
	}

	response, err := dc.cli.ImageBuild(dc.ctx, buildContext, buildOptions)
	if err != nil {
		return TerminalResponse{}, fmt.Errorf("failed to build image: %w", err)
	}

	// Show build output
	io.Copy(os.Stdout, response.Body)
	defer response.Body.Close()

	startResponse, err := dc.StartContainer(imageName+":latest", false)
	if err != nil {
		return TerminalResponse{}, fmt.Errorf("failed to start container: %w", err)
	}

	// Show container output
	go func() {
		io.Copy(os.Stdout, startResponse.Result)
	}()

	return startResponse, nil
}

func (dc *DockerClient) StartContainer(imageName string, pull bool) (TerminalResponse, error) {
	if pull {
		out, err := dc.cli.ImagePull(dc.ctx, imageName, image.PullOptions{})
		if err != nil {
			return TerminalResponse{}, fmt.Errorf("failed to pull image: %w", err)
		}
		io.Copy(io.Discard, out)
		defer out.Close()
	}

	resp, err := dc.cli.ContainerCreate(dc.ctx, &container.Config{
		Image: imageName,
	}, nil, nil, nil, "")
	if err != nil {
		return TerminalResponse{}, fmt.Errorf("failed to create container: %w", err)
	}

	if err := dc.cli.ContainerStart(dc.ctx, resp.ID, container.StartOptions{}); err != nil {
		return TerminalResponse{}, fmt.Errorf("failed to start container: %w", err)
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}

	logs, err := dc.cli.ContainerLogs(dc.ctx, resp.ID, options)
	if err != nil {
		return TerminalResponse{}, fmt.Errorf("failed to get container logs: %w", err)
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

// StopContainer stops a running container
func (dc *DockerClient) StopContainer(id string) error {
	return dc.cli.ContainerStop(dc.ctx, id, container.StopOptions{})
}

// RemoveContainer removes a container
func (dc *DockerClient) RemoveContainer(id string) error {
	return dc.cli.ContainerRemove(dc.ctx, id, container.RemoveOptions{})
}

// BuildAndStartContainerFromGitHub clones a GitHub repository and builds a Docker container from it
func (dc *DockerClient) BuildAndStartContainerFromGitHub(imageName string, githubURL string) (TerminalResponse, error) {
	// Create a git client
	gitClient, err := github.NewGitClient("")
	if err != nil {
		return TerminalResponse{}, fmt.Errorf("failed to create git client: %w", err)
	}

	// Clone the repository
	repoPath, err := gitClient.CloneRepository(githubURL)
	if err != nil {
		return TerminalResponse{}, fmt.Errorf("failed to clone repository: %w", err)
	}
	defer gitClient.CleanupRepository(repoPath) // Clean up after ourselves

	// Find Dockerfile in the cloned repository
	dockerfilePath, err := gitClient.FindDockerfile(repoPath)
	if err != nil {
		return TerminalResponse{}, fmt.Errorf("failed to find Dockerfile in %s: %w", repoPath, err)
	}

	localCodePath := "code_context.tar.gz"
	localCodeFile, err := os.Create(localCodePath)
	if err != nil {
		return TerminalResponse{}, fmt.Errorf("failed to create local code file %s: %w", localCodePath, err)
	}
	defer localCodeFile.Close() // Clean up after ourselves

	// Create pipe for tar stream
	pr, pw := io.Pipe()

	multiWriter := io.MultiWriter(localCodeFile, pw)
	tarErrChan := make(chan error, 1)

	// Stream the build context (which is the directory of the Dockerfile)
	go func() {
		var tarErr error
		defer func() {
			pw.CloseWithError(tarErr)
			tarErrChan <- tarErr
		}()

		// Using the multiWriter, we can write to both the local file and the pipe
		tarErr = gitClient.PrepareDockerBuildContext(dockerfilePath, multiWriter)
		if tarErr != nil {
			fmt.Fprintf(os.Stderr, "Error preparing Docker build context from %s: %v\n", filepath.Dir(dockerfilePath), tarErr)
		}
	}()

	// Build the image
	buildOptions := types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: filepath.Base(dockerfilePath), // Dockerfile name relative to the context (its own dir)
		Remove:     true,
	}

	response, buildErr := dc.cli.ImageBuild(dc.ctx, pr, buildOptions)
	tarringErr := <-tarErrChan // Wait for the tarring goroutine to finish and get its error status

	if tarringErr != nil {
		return TerminalResponse{}, fmt.Errorf("failed to prepare and write Docker build context: %w (docker build error: %v)", tarringErr, buildErr)
	}

	if buildErr != nil {
		return TerminalResponse{}, fmt.Errorf("failed to build image using Dockerfile %s: %w", dockerfilePath, buildErr)
	}

	fmt.Println("Image built successfully to ", imageName)

	// Show build output
	io.Copy(os.Stdout, response.Body)
	defer response.Body.Close()

	// Start the container
	startResponse, err := dc.StartContainer(imageName, false)
	if err != nil {
		return TerminalResponse{}, fmt.Errorf("failed to start container %s: %w", imageName, err)
	}

	// Show container output
	go func() {
		io.Copy(os.Stdout, startResponse.Result)
	}()

	return startResponse, nil
}

// example usage
// func main() {
// 	dc, err := CreateDockerClient()
// 	if err != nil {
// 		panic(err)
// 	}

// 	dc.StartContainer("hello-world")
// }
