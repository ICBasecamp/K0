package container

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// JobType represents the different types of container operations that can be queued
type JobType string

const (
	JobTypeCreate  JobType = "create"  // Create a new container
	JobTypeStop    JobType = "stop"    // Stop a running container
	JobTypeRemove  JobType = "remove"  // Remove a container
	JobTypeExecute JobType = "execute" // Execute a command in a container
)

// Job represents a container operation to be processed
type Job struct {
	Type             JobType          // Type of operation to perform
	ClientID         string           // ID of the client requesting the operation
	ImageName        string           // Name of the Docker image to use
	ContainerID      string           // ID of the container to operate on
	BuildContextPath string           // Path to the build context directory
	Result           chan interface{} // Channel to receive operation results
	Error            chan error       // Channel to receive operation errors
	CreatedAt        time.Time        // When the job was created
}

// JobQueue manages a pool of workers that process container operations
type JobQueue struct {
	jobs    chan *Job          // Channel for incoming jobs
	manager *ContainerManager  // Reference to the container manager
	workers int                // Number of worker goroutines
	wg      sync.WaitGroup     // WaitGroup to track worker goroutines
	ctx     context.Context    // Context for graceful shutdown
	cancel  context.CancelFunc // Function to cancel the context
}

// NewJobQueue creates a new job queue with the specified number of workers
func NewJobQueue(manager *ContainerManager, workers int) *JobQueue {
	ctx, cancel := context.WithCancel(context.Background())
	q := &JobQueue{
		jobs:    make(chan *Job, 100), // Buffer of 100 jobs
		manager: manager,
		workers: workers,
		ctx:     ctx,
		cancel:  cancel,
	}
	q.Start()
	return q
}

// Start initializes the worker goroutines
func (q *JobQueue) Start() {
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker()
	}
}

// Stop gracefully shuts down the job queue and its workers
func (q *JobQueue) Stop() {
	q.cancel()
	q.wg.Wait()
}

// worker processes jobs from the queue
func (q *JobQueue) worker() {
	defer q.wg.Done()

	for {
		select {
		case <-q.ctx.Done():
			return
		case job := <-q.jobs:
			var result interface{}
			var err error

			// Process different types of jobs
			switch job.Type {
			case JobTypeCreate:
				// Create a new container
				container, err := q.manager.CreateContainer(job.ClientID, job.ImageName, job.BuildContextPath)
				if err != nil {
					job.Error <- err
					continue
				}
				result = container
			case JobTypeStop:
				// Stop a container
				err = q.manager.StopContainer(job.ContainerID)
			case JobTypeRemove:
				// Remove a container
				err = q.manager.RemoveContainer(job.ContainerID)
			case JobTypeExecute:
				// Get container results
				container, err := q.manager.GetContainer(job.ContainerID)
				if err != nil {
					job.Error <- err
					continue
				}
				result = container.Result
			}

			// Handle errors
			if err != nil {
				job.Error <- err
				continue
			}

			// Send results back to the client
			if job.Result != nil {
				job.Result <- result
			}
		}
	}
}

// Enqueue adds a new job to the queue with a timeout
func (q *JobQueue) Enqueue(job *Job) error {
	select {
	case q.jobs <- job:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("job queue is full")
	}
}
