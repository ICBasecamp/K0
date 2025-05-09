package main

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/ICBasecamp/K0/backend/pkg/container"
	"github.com/ICBasecamp/K0/backend/pkg/docker"
	"github.com/ICBasecamp/K0/backend/pkg/s3"
)

func main() {
	// Create S3 client
	s3c, err := s3.CreateS3Client()
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}

	// Create Docker client
	dc, err := docker.CreateDockerClient(s3c)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}

	// Upload test Dockerfiles to S3 if they don't exist
	testImages := []string{"test-image-1", "test-image-2", "test-image-3"}
	for _, image := range testImages {
		s3Key := fmt.Sprintf("dockerfiles/%s.tar", image)

		// Check if file exists in S3
		exists, err := s3c.FileExists(s3Key)
		if err != nil {
			log.Printf("Error checking file existence: %v", err)
			continue
		}

		if exists {
			log.Printf("File %s already exists in S3, skipping upload", s3Key)
			continue
		}

		buildContextPath := filepath.Join("test-script", image)
		if err := s3c.TarAndUploadToS3(s3Key, buildContextPath); err != nil {
			log.Printf("Failed to upload %s to S3: %v", image, err)
			continue
		}
		log.Printf("Successfully uploaded %s to S3", image)
	}

	// Create container manager
	cm := container.NewContainerManager(dc, s3c)

	// Create job queue with 5 workers
	queue := container.NewJobQueue(cm, 5)
	defer queue.Stop()

	// Simulate multiple clients
	clients := []string{"client1", "client2", "client3"}
	images := []string{"test-image-1", "test-image-2", "test-image-3"}

	// Create containers for each client
	for i, clientID := range clients {
		// Create a job to create a container
		s3Key := fmt.Sprintf("dockerfiles/%s.tar", images[i])

		job := &container.Job{
			Type:             container.JobTypeCreate,
			ClientID:         clientID,
			ImageName:        images[i],
			BuildContextPath: s3Key,
			Result:           make(chan interface{}, 1),
			Error:            make(chan error, 1),
			CreatedAt:        time.Now(),
		}

		// Enqueue the job
		if err := queue.Enqueue(job); err != nil {
			log.Printf("Failed to enqueue job for client %s: %v", clientID, err)
			continue
		}

		// Wait for result
		select {
		case result := <-job.Result:
			container := result.(*container.Container)
			fmt.Printf("Client %s: Container created with ID %s\n", clientID, container.ID)
		case err := <-job.Error:
			log.Printf("Client %s: Error creating container: %v", clientID, err)
		case <-time.After(10 * time.Second):
			log.Printf("Client %s: Timeout waiting for container creation", clientID)
		}
	}

	// Wait a bit to see the containers running
	time.Sleep(5 * time.Second)

	// Stop all containers
	for _, clientID := range clients {
		containers := cm.ListClientContainers(clientID)
		for _, c := range containers {
			job := &container.Job{
				Type:        container.JobTypeStop,
				ClientID:    clientID,
				ContainerID: c.ID,
				Error:       make(chan error, 1),
				CreatedAt:   time.Now(),
			}

			if err := queue.Enqueue(job); err != nil {
				log.Printf("Failed to enqueue stop job for container %s: %v", c.ID, err)
				continue
			}

			select {
			case err := <-job.Error:
				log.Printf("Error stopping container %s: %v", c.ID, err)
			case <-time.After(5 * time.Second):
				log.Printf("Timeout stopping container %s", c.ID)
			}
		}
	}
}
