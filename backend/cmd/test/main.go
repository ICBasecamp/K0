package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ICBasecamp/K0/internal/container"
	"github.com/ICBasecamp/K0/internal/docker"
	"github.com/ICBasecamp/K0/internal/s3"
)

func main() {
	// Check if Docker is running
	if err := checkDockerRunning(); err != nil {
		log.Fatalf("Docker is not running: %v", err)
	}

	// Create S3 client
	fmt.Println("Initializing S3 client...")
	s3c, err := s3.CreateS3Client()
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}

	// Create Docker client
	fmt.Println("Initializing Docker client...")
	dc, err := docker.CreateDockerClient(s3c)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}

	// Create container manager
	fmt.Println("Creating container manager...")
	cm := container.NewContainerManager(dc, s3c)

	// First, ensure test image is uploaded to S3
	fmt.Println("\nChecking test image in S3...")
	s3Key := "dockerfiles/test-image-1.tar"
	exists, err := s3c.FileExists(s3Key)
	if err != nil {
		log.Fatalf("Failed to check file existence in S3: %v", err)
	}

	if !exists {
		fmt.Println("Test image not found in S3. Please run the server first to upload test images.")
		fmt.Println("Run: go run cmd/server/main.go")
		os.Exit(1)
	}

	// Test container creation
	fmt.Println("\nCreating a test container...")
	testContainer, err := cm.CreateContainer("test-client", "test-image-1", s3Key)
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}
	fmt.Printf("Container created successfully! ID: %s\n", testContainer.ID)

	// Test getting container
	fmt.Println("\nGetting container details...")
	container, err := cm.GetContainer(testContainer.ID)
	if err != nil {
		log.Fatalf("Failed to get container: %v", err)
	}
	fmt.Printf("Container details:\n")
	fmt.Printf("  ID: %s\n", container.ID)
	fmt.Printf("  ClientID: %s\n", container.ClientID)
	fmt.Printf("  ImageName: %s\n", container.ImageName)
	fmt.Printf("  Status: %s\n", container.Status)
	fmt.Printf("  CreatedAt: %s\n", container.CreatedAt)
	fmt.Printf("  UpdatedAt: %s\n", container.UpdatedAt)

	// Wait a bit to see the container running
	fmt.Println("\nWaiting for container to run...")
	time.Sleep(5 * time.Second)

	// Test stopping container
	fmt.Println("\nStopping container...")
	if err := cm.StopContainer(testContainer.ID); err != nil {
		log.Fatalf("Failed to stop container: %v", err)
	}
	fmt.Println("Container stopped successfully!")

	// Test removing container
	fmt.Println("\nRemoving container...")
	if err := cm.RemoveContainer(testContainer.ID); err != nil {
		log.Fatalf("Failed to remove container: %v", err)
	}
	fmt.Println("Container removed successfully!")

	// Test listing containers for a client
	fmt.Println("\nListing containers for test-client...")
	containers := cm.ListClientContainers("test-client")
	fmt.Printf("Found %d containers for test-client\n", len(containers))
}

func checkDockerRunning() error {
	// Try to connect to Docker daemon
	_, err := docker.CreateDockerClient(nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Docker daemon: %v", err)
	}
	return nil
}
