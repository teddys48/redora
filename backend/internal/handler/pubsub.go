package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/redora/redora/backend/internal/models"
	"github.com/redora/redora/backend/internal/service"
)

type PubSubHandler struct {
	svc service.RedisService
}

func NewPubSubHandler(svc service.RedisService) *PubSubHandler {
	return &PubSubHandler{svc: svc}
}

func (h *PubSubHandler) RegisterRoutes(router fiber.Router) {
	group := router.Group("/connections/:id/pubsub")
	group.Get("/channels", h.ListChannels)
	group.Post("/publish", h.PublishMessage)
}

func (h *PubSubHandler) ListChannels(c *fiber.Ctx) error {
	connID := c.Params("id")
	pattern := c.Query("pattern", "*")

	channels, err := h.svc.ListPubSubChannels(c.UserContext(), connID, pattern)
	if err != nil {
		return models.SendError(c, fiber.StatusInternalServerError, "PUBSUB_CHANNELS_FAILED", err.Error())
	}

	if channels == nil {
		channels = []string{}
	}

	return c.JSON(fiber.Map{"channels": channels})
}

func (h *PubSubHandler) PublishMessage(c *fiber.Ctx) error {
	connID := c.Params("id")

	var input models.PubSubPublishInput
	if err := c.BodyParser(&input); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid JSON payload")
	}

	subscribers, err := h.svc.PublishMessage(c.UserContext(), connID, input.Channel, input.Message)
	if err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "PUBLISH_FAILED", err.Error())
	}

	return c.JSON(fiber.Map{
		"message":     "Message published successfully",
		"subscribers": subscribers,
	})
}
