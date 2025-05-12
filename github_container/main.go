// cmd/test/main.go
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ICBasecamp/K0/backend/pkg/container"
	"github.com/ICBasecamp/K0/backend/pkg/docker"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: test <image-name> <github-url>")
		fmt.Println("Example: test my-app https://github.com/username/repo")
		os.Exit(1)
	}

	imageName := os.Args[1]
	githubURL := os.Args[2]

	// Create Docker client
	dockerClient, err := docker.CreateDockerClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer dockerClient.Cleanup()

	// Create container manager
	containerManager := container.NewContainerManager(dockerClient)

	// Create container from GitHub
	container, err := containerManager.CreateContainerFromGitHub("test-client", imageName, githubURL)
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}
	log.Printf("Container created with ID: %s", container.ID)

	// Let it run for a while
	time.Sleep(30 * time.Second)

	// Clean up
	if err := containerManager.StopContainer(container.ID); err != nil {
		log.Printf("Failed to stop container: %v", err)
	}

	if err := containerManager.RemoveContainer(container.ID); err != nil {
		log.Printf("Failed to remove container: %v", err)
	}
}
