package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"sync"

	"github.com/ICBasecamp/K0/backend/internal/github"
	"github.com/ICBasecamp/K0/backend/pkg/ec2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

type DockerClient struct {
	cli        *client.Client
	ctx        context.Context
	ec2Client  *ec2.EC2Client
	instanceID string
	publicIP   string
}

type TerminalResponse struct {
	ID     string
	Result io.ReadCloser
}

func getAmiIdFromSSM(ctx context.Context, ssmClient *ssm.Client) (string, error) {
	param, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name: aws.String("/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-x86_64-gp2"),
	})
	if err != nil {
		return "", err
	}
	return *param.Parameter.Value, nil
}

func CreateDockerClient() (*DockerClient, error) {
	// Check if we should use local Docker (no AWS)
	useLocalDocker := os.Getenv("USE_LOCAL_DOCKER")
	if useLocalDocker == "true" {
		return createLocalDockerClient()
	}
	
	// Default to AWS EC2 setup
	return createEC2DockerClient()
}

func createLocalDockerClient() (*DockerClient, error) {
	fmt.Println("Using local Docker daemon...")
	
	// Create Docker client that connects to local Docker daemon
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create local Docker client: %w", err)
	}

	// Try to ping the Docker daemon
	_, err = cli.Ping(context.Background())
	if err != nil {
		cli.Close()
		return nil, fmt.Errorf("failed to connect to local Docker daemon. Make sure Docker Desktop is running: %w", err)
	}

	fmt.Println("Successfully connected to local Docker daemon!")

	return &DockerClient{
		cli:        cli,
		ctx:        context.Background(),
		ec2Client:  nil, // No EC2 client needed for local
		instanceID: "",  // No instance ID for local
		publicIP:   "",  // No public IP for local
	}, nil
}

func createEC2DockerClient() (*DockerClient, error) {
	ec2Client, err := ec2.NewEC2Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	// Dynamically fetch the latest Amazon Linux 2 AMI ID for the current region
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for SSM: %w", err)
	}
	ssmClient := ssm.NewFromConfig(cfg)
	amiID, err := getAmiIdFromSSM(context.Background(), ssmClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest Amazon Linux 2 AMI ID: %w", err)
	}

	// Create instance and wait for it to be ready
	instanceId, err := ec2Client.CreateInstance(
		"Docker-Sandbox",
		amiID,
		"t3.micro",
		"subnet-0988a47a3010e4968", // your subnet ID
		"sg-042b1651131eb71d2",     // your security group ID
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	// Get instance details to get the public IP
	describeResult, err := ec2Client.DescribeInstance(instanceId)
	if err != nil {
		ec2Client.TerminateInstance(instanceId)
		return nil, fmt.Errorf("failed to get instance IP: %w", err)
	}
	publicIP := *describeResult.Reservations[0].Instances[0].PublicIpAddress

	// Get system logs to check Docker status
	logs, err := ec2Client.GetInstanceLogs(instanceId)
	if err != nil {
		fmt.Printf("Warning: Failed to get system logs: %v\n", err)
	} else {
		fmt.Println("Checking system logs for Docker status...")
		fmt.Println(logs)
	}

	fmt.Printf("Attempting to connect to Docker daemon at %s:2375...\n", publicIP)

	// Create Docker client that connects to remote instance
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithHost(fmt.Sprintf("tcp://%s:2375", publicIP)),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		ec2Client.TerminateInstance(instanceId)
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Try to ping the Docker daemon
	_, err = cli.Ping(context.Background())
	if err != nil {
		cli.Close()
		ec2Client.TerminateInstance(instanceId)
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	fmt.Println("Successfully connected to Docker daemon!")

	return &DockerClient{
		cli:        cli,
		ctx:        context.Background(),
		ec2Client:  ec2Client,
		instanceID: instanceId,
		publicIP:   publicIP,
	}, nil
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

// Removed unused debugging functions: PrintTerminalResponse and ListImages

// StopContainer stops a running container
func (dc *DockerClient) StopContainer(id string) error {
	return dc.cli.ContainerStop(dc.ctx, id, container.StopOptions{})
}

// RemoveContainer removes a container
func (dc *DockerClient) RemoveContainer(id string) error {
	return dc.cli.ContainerRemove(dc.ctx, id, container.RemoveOptions{})
}

// Removed BuildAndStartContainerFromGitHub - only used by deprecated container manager

func (dc *DockerClient) Cleanup() error {
	if dc.ec2Client != nil && dc.instanceID != "" {
		return dc.ec2Client.TerminateInstance(dc.instanceID)
	}
	return nil
}

// BuildAndStartContainerFromGitHubWS builds a Docker container from a GitHub repository and starts/returns a websocket connection to the container output
func (dc *DockerClient) BuildAndStartContainerFromGitHubWS(imageName string, githubURL string, ContainerStreams *sync.Map) (TerminalResponse, error) {
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

	// fmt.Println("Image built successfully to ", imageName)

	// Show build output
	// io.Copy(os.Stdout, response.Body)
	defer response.Body.Close()

	// Start the container
	startResponse, err := dc.StartContainer(imageName, false)
	if err != nil {
		return TerminalResponse{}, fmt.Errorf("failed to start container %s: %w", imageName, err)
	}

	fmt.Println("starting container output to image name / websocket connection name: ", imageName)

	// Show container output
	go func() {
		// gotta return this somehow instead of printing
		ContainerStreams.Store(imageName, startResponse.Result)
	}()

	return startResponse, nil
}

// Removed commented S3 and example code
