package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/redora/redora/backend/internal/models"
	"github.com/redora/redora/backend/internal/service"
)

type CommandHandler struct {
	svc service.RedisService
}

func NewCommandHandler(svc service.RedisService) *CommandHandler {
	return &CommandHandler{svc: svc}
}

func (h *CommandHandler) RegisterRoutes(router fiber.Router) {
	router.Post("/connections/:id/command", h.ExecuteCommand)
}

func (h *CommandHandler) ExecuteCommand(c *fiber.Ctx) error {
	connID := c.Params("id")

	var input models.ExecuteCommandInput
	if err := c.BodyParser(&input); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid JSON payload")
	}

	res, err := h.svc.ExecuteCommand(c.UserContext(), connID, input.Command)
	if err != nil {
		return models.SendError(c, fiber.StatusInternalServerError, "COMMAND_EXECUTION_FAILED", err.Error())
	}

	return c.JSON(res)
}
