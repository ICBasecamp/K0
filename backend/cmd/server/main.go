package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/ICBasecamp/K0/backend/pkg/docker"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"

	"github.com/joho/godotenv"
	"github.com/supabase-community/supabase-go"
)

var (
	ContainerStreams = sync.Map{}
	supabaseClient   *supabase.Client
	dockerClient     *docker.DockerClient
)

func main() {
	log.Println("Starting K0 backend server...")
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}
	log.Println("Environment file loaded successfully")

	// S3 client removed - no longer needed for simplified Docker service

	// Create Docker client
	log.Println("Creating Docker client...")
	dockerClient, err = docker.CreateDockerClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	log.Println("Docker client created successfully")

	// Create supabase client
	log.Println("Initializing Supabase client...")
	var supabaseErr error
	supabaseClient, supabaseErr = supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_ANON_KEY"), &supabase.ClientOptions{})
	if supabaseErr != nil {
		log.Fatalf("Failed to initialize Supabase client: %v", supabaseErr)
	}
	log.Println("Supabase client initialized successfully")

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

		// Create container directly using Docker client - simplified for microservice architecture
		response, err := dockerClient.BuildAndStartContainerFromGitHubWS(imageName, requestBody.GitHubLink, &ContainerStreams)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to create container: %v", err),
			})
		}

		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("Starting GitHub container...")
		log.Printf("Container created successfully with ID: %s", response.ID)

		// Sleep for 5 seconds to allow container to start up
		time.Sleep(5 * time.Second)

		// Return the WebSocket connection name immediately
		return c.JSON(fiber.Map{
			"ws_connection_name": imageName,
			"container_id":       response.ID,
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

	// encode websocket connection id and room id by separating with "___"
	app.Get("/ws/container-output/:id", websocket.New(func(c *websocket.Conn) {
		params := c.Params("id")
		parts := strings.Split(params, "___")
		if len(parts) != 2 {
			c.WriteMessage(websocket.TextMessage, []byte("Invalid parameters"))
			return
		}
		id := parts[0]
		roomId := parts[1]
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

			currentTerminalOutput, _, err := supabaseClient.From("running_rooms").Select("terminal_output", "", false).Eq("id", roomId).Single().Execute()
			if err != nil {
				log.Printf("Error getting terminal output: %v", err)
			}
			var result struct {
				TerminalOutput string `json:"terminal_output"`
			}
			if err := json.Unmarshal(currentTerminalOutput, &result); err != nil {
				log.Printf("Error parsing terminal output: %v", err)
				continue
			}
			newOutput := result.TerminalOutput + filterPrintable(buf[:n])

			// update running_rooms table with terminal_output
			_, _, err = supabaseClient.From("running_rooms").Update(
				map[string]any{"terminal_output": newOutput},
				"",
				"",
			).Eq("id", roomId).Execute() // Use the actual room ID from the connection

			if err != nil {
				log.Printf("Error updating terminal output: %v", err)
			}
		}
	}))

	log.Println("Starting server on port 3009...")
	if err := app.Listen(":3009"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func filterPrintable(input []byte) string {
	out := make([]rune, 0, len(input))
	for _, r := range string(input) {
		if unicode.IsPrint(r) || r == '\n' || r == '\r' || r == '\t' {
			out = append(out, r)
		}
	}
	return string(out)
}
