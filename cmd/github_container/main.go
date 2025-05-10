package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ICBasecamp/K0/backend/pkg/container"
	"github.com/ICBasecamp/K0/backend/pkg/docker"
	"github.com/ICBasecamp/K0/backend/pkg/s3"
)

// A simple test program to demonstrate creating a container from a GitHub repository
func main() {
	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting GitHub container test...")

	if len(os.Args) < 3 {
		log.Println("Error: Missing required arguments")
		fmt.Println("Usage: github_container [image-name] [github-url]")
		fmt.Println("Example: github_container my-app https://github.com/example/docker-app")
		os.Exit(1)
	}

	imageName := os.Args[1]
	githubURL := os.Args[2]
	log.Printf("Image name: %s", imageName)
	log.Printf("GitHub URL: %s", githubURL)

	// Create the S3 client
	log.Println("Creating S3 client...")
	s3Client, err := s3.CreateS3Client()
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}
	log.Println("S3 client created successfully")

	// Create the Docker client
	log.Println("Creating Docker client...")
	dockerClient, err := docker.CreateDockerClient(s3Client)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	log.Println("Docker client created successfully")

	// Create the container manager
	log.Println("Creating container manager...")
	containerManager := container.NewContainerManager(dockerClient, s3Client)
	log.Println("Container manager created successfully")

	// Create a container from the GitHub repository
	log.Printf("Creating container from GitHub repository: %s", githubURL)
	container, err := containerManager.CreateContainerFromGitHub("test-client", imageName, githubURL)
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}
	log.Printf("Container created successfully with ID: %s", container.ID)

	// Give the container some time to run
	log.Println("Waiting for container to run...")
	time.Sleep(30 * time.Second)

	// Stop and remove the container
	log.Println("Stopping container...")
	if err := containerManager.StopContainer(container.ID); err != nil {
		log.Fatalf("Failed to stop container: %v", err)
	}
	log.Println("Container stopped successfully")

	log.Println("Removing container...")
	if err := containerManager.RemoveContainer(container.ID); err != nil {
		log.Fatalf("Failed to remove container: %v", err)
	}
	log.Println("Container removed successfully")

	log.Println("Test completed successfully!")
}
