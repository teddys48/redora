package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

type HealthHandler struct {
	version string
}

func NewHealthHandler(version string) *HealthHandler {
	if version == "" {
		version = "1.0.0"
	}
	return &HealthHandler{version: version}
}

func (h *HealthHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/health", h.CheckHealth)
}

func (h *HealthHandler) CheckHealth(c *fiber.Ctx) error {
	return c.JSON(HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   h.version,
	})
}
