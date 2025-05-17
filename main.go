package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// Config represents the application configuration settings.
type Config struct {
	Env      string
	Endpoint string
	Port     string
}

// Env is a type used for loading and managing environment-specific configuration settings.
type Env struct{}

// Load retrieves the application configuration by reading environment variables or using default values.
func (l *Env) Load() Config {
	env := getEnvOr("APP_ENV", "development")
	endpoint := getEnvOr("ENDPOINT", "http://0.0.0.0")
	port := getEnvOr("PORT", "8080")

	return Config{
		Env:      env,
		Endpoint: endpoint,
		Port:     port,
	}
}

func getEnvOr(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Router defines an interface for setting up application routes with a given Fiber app and configuration.
type Router interface {
	SetupRoutes(app *fiber.App, config Config)
}

// APIRouter is a struct used for setting up routes in a Fiber application.
type APIRouter struct{}

// SetupRoutes registers routes for the application, including root, info, and health endpoints, using the provided configuration.
func (r *APIRouter) SetupRoutes(app *fiber.App, config Config) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello Pyment!")
	})

	app.Get("/info", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"env":      config.Env,
			"port":     config.Port,
			"endpoint": fmt.Sprintf("%s:%s", config.Endpoint, config.Port),
		})
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
}

// Server represents an HTTP server instance with application configuration and routing.
type Server struct {
	app    *fiber.App
	config Config
}

// NewServer initializes a new Server instance with the provided Config and Router and sets up routing for the application.
func NewServer(config Config, router Router) *Server {
	app := fiber.New()
	app.Use(logger.New())

	router.SetupRoutes(app, config)

	return &Server{
		app:    app,
		config: config,
	}
}

// Start begins the server by binding it to the configured port and environment. Logs the start status and runs asynchronously.
func (s *Server) Start() {
	endpoint := fmt.Sprintf("%s:%s", s.config.Endpoint, s.config.Port)
	log.Printf("Server starting on %s (Environment: %s)", endpoint, s.config.Env)

	go func() {
		if err := s.app.Listen(":" + s.config.Port); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	}()
}

// Shutdown gracefully stops the server, ensuring all connections are closed within a timeout of 5 seconds.
func (s *Server) Shutdown() {
	log.Println("Shutting down server...")

	if err := s.app.ShutdownWithTimeout(5 * time.Second); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server shutdown gracefully")
}

func main() {
	env := &Env{}
	router := &APIRouter{}

	config := env.Load()

	server := NewServer(config, router)
	server.Start()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interrupt

	server.Shutdown()
}
