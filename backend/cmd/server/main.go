package main

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/ICBasecamp/K0/backend/pkg/container"
	"github.com/ICBasecamp/K0/backend/pkg/docker"
	"github.com/ICBasecamp/K0/backend/pkg/s3"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
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

	// Create container manager
	cm := container.NewContainerManager(dc, s3c)

	var ContainerStreams = sync.Map{}

	app := fiber.New(fiber.Config{
		ReadBufferSize:  1024 * 1024, 
		WriteBufferSize: 1024 * 1024,
		BodyLimit:       10 * 1024 * 1024,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:3000",
		AllowHeaders: "Origin, Content-Type, Accept, Connection, Upgrade",
		AllowMethods: "GET, POST, OPTIONS",
	}))

	app.Use(func(c *fiber.Ctx) error {
		log.Println("➡️", c.Method(), c.Path())
		return c.Next()
	})

	app.Use("/ws/*", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Set("Access-Control-Allow-Origin", "http://localhost:3000")
			c.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Connection, Upgrade")
			return c.Next()
		}
		return c.Next()
	})

	// test ws connection
	app.Get("/ws/test", websocket.New(func(c *websocket.Conn) {
		defer c.Close()
		log.Println("✅ WS connected to /ws/test")

		for {
			time.Sleep(time.Second)
			if err := c.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
				log.Println("Write error:", err)
				break
			}
		}
	}))
	

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("testing")
	})

	app.Post("/start-github-container", func(c *fiber.Ctx) error {
		type RequestBody struct {
			RoomID     string `json:"room_id"`
			GitHubLink string `json:"github_link"`
		}

		var requestBody RequestBody
		if err := c.BodyParser(&requestBody); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if requestBody.RoomID == "" || requestBody.GitHubLink == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Room ID and GitHub link are required",
			})
		}

		// create unique image name based on room id and timestamp
		// we use imagename as ws connection name, but container id is still required for stopping and removing the container
		imageName := fmt.Sprintf("github-container-%s-%d", requestBody.RoomID, time.Now().Unix())

		container, err := cm.CreateContainerFromGitHubWS(requestBody.RoomID, imageName, requestBody.GitHubLink, &ContainerStreams)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to create container: %v", err),
			})
		}

		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("Starting GitHub container...")
		log.Printf("Container created successfully with ID: %s", container.ID)

		// Sleep for 5 seconds to allow container to start up
		time.Sleep(5 * time.Second)

		// Return the WebSocket connection name immediately
		return c.JSON(fiber.Map{
			"ws_connection_name": imageName,
			"container_id":       container.ID,
		})
	})


	app.Use("/ws/container-output/:id", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Set("Access-Control-Allow-Origin", "*") // Allow frontend origin here
			c.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept")
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws/container-output/:id", websocket.New(func(c *websocket.Conn) {
		id := c.Params("id")
		log.Println("Websocket connection established for ID:", id)

		streamRaw, ok := ContainerStreams.Load(id)
		if !ok {
			c.WriteMessage(websocket.TextMessage, []byte("Invalid container ID"))
			return
		}

		stream := streamRaw.(io.ReadCloser)
		defer stream.Close()

		buf := make([]byte, 1024)
		for {
			n, err := stream.Read(buf)
			if err != nil {
				break
			}
			if writeErr := c.WriteMessage(websocket.TextMessage, buf[:n]); writeErr != nil {
				break
			}
		}

	}))

	app.Listen(":3009")
}
