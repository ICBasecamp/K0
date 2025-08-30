// DEPRECATED: This package is deprecated in favor of using docker.DockerClient directly
// for a simplified microservice architecture. Use docker.BuildAndStartContainerFromGitHubWS
// and other docker methods directly instead of this container management layer.
package container

import (
	"fmt"
	"sync"
	"time"

	"github.com/ICBasecamp/K0/backend/pkg/docker"
)

// Container represents a running Docker container with its metadata
type Container struct {
	ID        string           // Unique identifier for the container
	ClientID  string           // ID of the client who owns this container
	ImageName string           // Name of the Docker image used
	Status    ContainerStatus  // Current status of the container
	Result    chan interface{} // Channel to receive container execution results
	CreatedAt time.Time        // When the container was created
	UpdatedAt time.Time        // Last time the container status was updated
}

// ContainerStatus represents the possible states of a container
type ContainerStatus string

const (
	StatusCreated ContainerStatus = "created" // Container was created but not started
	StatusRunning ContainerStatus = "running" // Container is currently running
	StatusStopped ContainerStatus = "stopped" // Container was stopped
	StatusRemoved ContainerStatus = "removed" // Container was removed
	StatusError   ContainerStatus = "error"   // Container encountered an error
)

// ContainerManager handles the lifecycle of Docker containers
type ContainerManager struct {
	dockerClient *docker.DockerClient
	containers   map[string]*Container
	mu           sync.RWMutex
}

// NewContainerManager creates a new container manager instance
func NewContainerManager(dockerClient *docker.DockerClient) *ContainerManager {
	return &ContainerManager{
		dockerClient: dockerClient,
		containers:   make(map[string]*Container),
	}
}

// Removed deprecated S3-based CreateContainer method

// GetContainer retrieves a container by its ID
func (cm *ContainerManager) GetContainer(id string) (*Container, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	container, exists := cm.containers[id]
	if !exists {
		return nil, fmt.Errorf("container not found: %s", id)
	}

	return container, nil
}

// StopContainer stops a running container
func (cm *ContainerManager) StopContainer(id string) error {
	container, err := cm.GetContainer(id)
	if err != nil {
		return err
	}

	// Stop the container using Docker client
	if err := cm.dockerClient.StopContainer(id); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Update container status
	cm.mu.Lock()
	container.Status = StatusStopped
	container.UpdatedAt = time.Now()
	cm.mu.Unlock()

	return nil
}

// RemoveContainer removes a container
func (cm *ContainerManager) RemoveContainer(id string) error {
	_, err := cm.GetContainer(id)
	if err != nil {
		return err
	}

	// Remove the container using Docker client
	if err := cm.dockerClient.RemoveContainer(id); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Remove container from our map
	cm.mu.Lock()
	delete(cm.containers, id)
	cm.mu.Unlock()

	return nil
}

// monitorContainerOutput monitors the output of a container and updates its status
func (cm *ContainerManager) monitorContainerOutput(container *Container) {
	// Read container logs
	for {
		select {
		case <-container.Result:
			// Update container status based on result
			cm.mu.Lock()
			container.Status = StatusStopped
			container.UpdatedAt = time.Now()
			cm.mu.Unlock()
			return
		}
	}
}

func (cm *ContainerManager) ListClientContainers(clientID string) []*Container {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var clientContainers []*Container
	for _, container := range cm.containers {
		if container.ClientID == clientID {
			clientContainers = append(clientContainers, container)
		}
	}

	return clientContainers
}

// Removed deprecated CreateContainerFromGitHub method - use CreateContainerFromGitHubWS instead

func (cm *ContainerManager) CreateContainerFromGitHubWS(clientID, imageName, githubURL string, ContainerStreams *sync.Map) (*Container, error) {
	// Start a container using the Docker client with GitHub repository
	response, err := cm.dockerClient.BuildAndStartContainerFromGitHubWS(imageName, githubURL, ContainerStreams)
	if err != nil {
		return nil, fmt.Errorf("failed to create container from GitHub: %w", err)
	}

	// Create a new Container struct to track the container
	container := &Container{
		ID:        response.ID,
		ClientID:  clientID,
		ImageName: imageName,
		Status:    StatusRunning,
		Result:    make(chan interface{}, 1),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store the container in our map
	cm.mu.Lock()
	cm.containers[container.ID] = container
	cm.mu.Unlock()

	// Start a goroutine to monitor container output
	go cm.monitorContainerOutput(container)

	return container, nil
}
