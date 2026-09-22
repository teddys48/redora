package handler_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/redora/redora/backend/internal/handler"
)

func TestHealthCheckHandler(t *testing.T) {
	app := fiber.New()
	healthHandler := handler.NewHealthHandler("1.0.0-test")
	apiGroup := app.Group("/api")
	healthHandler.RegisterRoutes(apiGroup)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var res handler.HealthResponse
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if res.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", res.Status)
	}
	if res.Version != "1.0.0-test" {
		t.Errorf("Expected version '1.0.0-test', got '%s'", res.Version)
	}
	if res.Timestamp == "" {
		t.Error("Expected non-empty timestamp")
	}
}
