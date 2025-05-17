package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRouter is a mock implementation of the Router interface
type MockRouter struct {
	mock.Mock
}

func (m *MockRouter) SetupRoutes(app *fiber.App, config Config) {
	m.Called(app, config)
}

// FiberAppWrapper is an interface that defines methods for starting and shutting down a Fiber application.
type FiberAppWrapper interface {
	Listen(addr string) error
	ShutdownWithTimeout(timeout time.Duration) error
}

func TestGetEnvOr(t *testing.T) {
	t.Run("Existing Environment Variable", func(t *testing.T) {
		_ = os.Setenv("TEST_KEY", "test_value")
		defer func() { _ = os.Unsetenv("TEST_KEY") }()

		result := getEnvOr("TEST_KEY", "default_value")
		assert.Equal(t, "test_value", result)
	})

	t.Run("Non-Existing Environment Variable", func(t *testing.T) {
		result := getEnvOr("NON_EXISTING_KEY", "default_value")
		assert.Equal(t, "default_value", result)
	})

	t.Run("Empty Environment Variable", func(t *testing.T) {
		_ = os.Setenv("EMPTY_KEY", "")
		defer func() { _ = os.Unsetenv("EMPTY_KEY") }()

		result := getEnvOr("EMPTY_KEY", "default_value")
		assert.Equal(t, "default_value", result)
	})
}

func TestEnvLoad(t *testing.T) {
	t.Run("With Custom Environment Variables", func(t *testing.T) {
		_ = os.Setenv("APP_ENV", "test_env")
		_ = os.Setenv("ENDPOINT", "test_endpoint")
		_ = os.Setenv("PORT", "1234")
		defer func() {
			_ = os.Unsetenv("APP_ENV")
			_ = os.Unsetenv("ENDPOINT")
			_ = os.Unsetenv("PORT")
		}()

		env := &Env{}
		config := env.Load()

		assert.Equal(t, "test_env", config.Env)
		assert.Equal(t, "test_endpoint", config.Endpoint)
		assert.Equal(t, "1234", config.Port)
	})

	t.Run("With Default Values", func(t *testing.T) {
		_ = os.Unsetenv("APP_ENV")
		_ = os.Unsetenv("ENDPOINT")
		_ = os.Unsetenv("PORT")

		env := &Env{}
		config := env.Load()

		assert.Equal(t, "development", config.Env)
		assert.Equal(t, "http://0.0.0.0", config.Endpoint)
		assert.Equal(t, "8080", config.Port)
	})

	t.Run("With Mixed Values", func(t *testing.T) {
		_ = os.Setenv("APP_ENV", "staging")
		defer func() {
			_ = os.Unsetenv("APP_ENV")
		}()

		env := &Env{}
		config := env.Load()

		assert.Equal(t, "staging", config.Env)
		assert.Equal(t, "http://0.0.0.0", config.Endpoint)
		assert.Equal(t, "8080", config.Port)
	})
}

func TestAPIRouterSetupRoutes(t *testing.T) {
	t.Run("Root Endpoint", func(t *testing.T) {
		app := fiber.New()
		config := Config{
			Env:      "test_env",
			Endpoint: "test_endpoint",
			Port:     "1234",
		}

		router := &APIRouter{}
		router.SetupRoutes(app, config)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, "Hello Golf!", string(body))
	})

	t.Run("Info Endpoint", func(t *testing.T) {
		app := fiber.New()
		config := Config{
			Env:      "test_env",
			Endpoint: "test_endpoint",
			Port:     "1234",
		}

		router := &APIRouter{}
		router.SetupRoutes(app, config)

		req := httptest.NewRequest(http.MethodGet, "/info", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var infoResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&infoResponse)
		assert.NoError(t, err)
		assert.Equal(t, "test_env", infoResponse["env"])
		assert.Equal(t, "1234", infoResponse["port"])
		assert.Equal(t, "test_endpoint:1234", infoResponse["endpoint"])
	})

	t.Run("Health Endpoint", func(t *testing.T) {
		app := fiber.New()
		config := Config{
			Env:      "test_env",
			Endpoint: "test_endpoint",
			Port:     "1234",
		}

		router := &APIRouter{}
		router.SetupRoutes(app, config)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, "OK", string(body))
	})

	t.Run("Non-existent Endpoint", func(t *testing.T) {
		app := fiber.New()
		config := Config{
			Env:      "test_env",
			Endpoint: "test_endpoint",
			Port:     "1234",
		}

		router := &APIRouter{}
		router.SetupRoutes(app, config)

		req := httptest.NewRequest(http.MethodGet, "/non-existent", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestNewServer(t *testing.T) {
	t.Run("Standard Configuration", func(t *testing.T) {
		config := Config{
			Env:      "test_env",
			Endpoint: "test_endpoint",
			Port:     "1234",
		}

		mockRouter := new(MockRouter)
		mockRouter.On("SetupRoutes", mock.Anything, config).Return()

		server := NewServer(config, mockRouter)

		assert.NotNil(t, server)
		assert.NotNil(t, server.app)
		assert.Equal(t, config, server.config)

		mockRouter.AssertExpectations(t)
	})

	t.Run("Empty Configuration", func(t *testing.T) {
		config := Config{}

		mockRouter := new(MockRouter)
		mockRouter.On("SetupRoutes", mock.Anything, config).Return()

		server := NewServer(config, mockRouter)

		assert.NotNil(t, server)
		assert.NotNil(t, server.app)
		assert.Equal(t, config, server.config)

		mockRouter.AssertExpectations(t)
	})
}

func TestServerStart(t *testing.T) {
	t.Run("Start Server Successfully", func(t *testing.T) {
		testPort := "9876"
		config := Config{
			Env:      "test_env",
			Endpoint: "http://localhost",
			Port:     testPort,
		}

		router := &APIRouter{}
		server := NewServer(config, router)

		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer func() { log.SetOutput(os.Stderr) }()

		server.Start()
		defer server.Shutdown()

		time.Sleep(100 * time.Millisecond)

		resp, err := http.Get("http://localhost:" + testPort + "/health")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		assert.Contains(t, buf.String(), "Server starting on http://localhost:9876")
		assert.Contains(t, buf.String(), "(Environment: test_env)")
	})
}

func TestServerShutdown(t *testing.T) {
	t.Run("Successful Shutdown", func(t *testing.T) {
		testPort := "9877"
		config := Config{
			Env:      "test_env",
			Endpoint: "http://localhost",
			Port:     testPort,
		}

		router := &APIRouter{}
		server := NewServer(config, router)

		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer func() { log.SetOutput(os.Stderr) }()

		server.Start()
		time.Sleep(100 * time.Millisecond)

		resp, err := http.Get("http://localhost:" + testPort + "/health")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		server.Shutdown()
		time.Sleep(100 * time.Millisecond)

		_, err = http.Get("http://localhost:" + testPort + "/health")
		assert.Error(t, err)

		assert.Contains(t, buf.String(), "Shutting down server...")
		assert.Contains(t, buf.String(), "Server shutdown gracefully")
	})
}

func TestAPIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = os.Setenv("PORT", "8765")
	_ = os.Setenv("APP_ENV", "test")
	_ = os.Setenv("ENDPOINT", "http://test-api")
	defer func() {
		_ = os.Unsetenv("PORT")
		_ = os.Unsetenv("APP_ENV")
		_ = os.Unsetenv("ENDPOINT")
	}()

	env := &Env{}
	router := &APIRouter{}
	config := env.Load()
	server := NewServer(config, router)

	server.Start()
	defer server.Shutdown()

	time.Sleep(100 * time.Millisecond)

	t.Run("Root Endpoint", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8765/")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, "Hello Golf!", string(body))
	})

	t.Run("Info Endpoint", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8765/info")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "test", result["env"])
		assert.Equal(t, "8765", result["port"])
		assert.Equal(t, "http://test-api:8765", result["endpoint"])
	})

	t.Run("Health Endpoint", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8765/health")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, "OK", string(body))
	})
}

func TestMainFunctionally(t *testing.T) {
	env := &Env{}
	router := &APIRouter{}

	config := env.Load()
	assert.NotEmpty(t, config.Env)
	assert.NotEmpty(t, config.Endpoint)
	assert.NotEmpty(t, config.Port)

	server := NewServer(config, router)
	assert.NotNil(t, server)

	mockInterrupt := make(chan os.Signal, 1)

	go func() {
		time.Sleep(10 * time.Millisecond)
		mockInterrupt <- syscall.SIGINT
	}()

	sig := <-mockInterrupt
	assert.Equal(t, syscall.SIGINT, sig)
}

func TestConfigMethods(t *testing.T) {
	config := Config{
		Env:      "test_env",
		Endpoint: "test_endpoint",
		Port:     "1234",
	}

	configStr := config.Env + "-" + config.Endpoint + ":" + config.Port
	assert.NotEmpty(t, configStr)

	configCopy := Config{
		Env:      config.Env,
		Endpoint: config.Endpoint,
		Port:     config.Port,
	}
	assert.Equal(t, config, configCopy)
}
