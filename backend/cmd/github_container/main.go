package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/ICBasecamp/K0/backend/pkg/docker"
	"github.com/joho/godotenv"
)

// A simple test program to demonstrate creating a container from a GitHub repository
func main() {
	// Load environment variables from .env file
	err := godotenv.Load("../../../.env")
	if err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}

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

	// Create the Docker client
	log.Println("Creating Docker client...")
	dockerClient, err := docker.CreateDockerClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	log.Println("Docker client created successfully")

	// Create a container from the GitHub repository directly using Docker client
	log.Printf("Creating container from GitHub repository: %s", githubURL)
	var containerStreams sync.Map
	response, err := dockerClient.BuildAndStartContainerFromGitHubWS(imageName, githubURL, &containerStreams)
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}
	log.Printf("Container created successfully with ID: %s", response.ID)

	// Give the container some time to run
	log.Println("Waiting for container to run...")
	time.Sleep(30 * time.Second)

	// Stop and remove the container directly
	log.Println("Stopping container...")
	if err := dockerClient.StopContainer(response.ID); err != nil {
		log.Fatalf("Failed to stop container: %v", err)
	}
	log.Println("Container stopped successfully")

	log.Println("Removing container...")
	if err := dockerClient.RemoveContainer(response.ID); err != nil {
		log.Fatalf("Failed to remove container: %v", err)
	}
	log.Println("Container removed successfully")

	log.Println("Test completed successfully!")
}
