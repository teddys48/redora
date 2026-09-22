package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/redora/redora/backend/internal/models"
	"github.com/redora/redora/backend/internal/service"
)

type ConnectionHandler struct {
	svc service.ConnectionService
}

func NewConnectionHandler(svc service.ConnectionService) *ConnectionHandler {
	return &ConnectionHandler{svc: svc}
}

func (h *ConnectionHandler) RegisterRoutes(router fiber.Router) {
	group := router.Group("/connections")
	group.Get("/", h.GetAll)
	group.Post("/", h.Create)
	group.Post("/test", h.TestInput)
	group.Get("/:id", h.GetByID)
	group.Put("/:id", h.Update)
	group.Delete("/:id", h.Delete)
	group.Post("/:id/test", h.TestByID)
}

func (h *ConnectionHandler) GetAll(c *fiber.Ctx) error {
	connections, err := h.svc.GetAll(c.UserContext())
	if err != nil {
		return models.SendError(c, fiber.StatusInternalServerError, "DB_QUERY_FAILED", err.Error())
	}
	if connections == nil {
		connections = []*models.Connection{}
	}
	return c.JSON(fiber.Map{"data": connections})
}

func (h *ConnectionHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	conn, err := h.svc.GetByID(c.UserContext(), id)
	if err != nil {
		return models.SendError(c, fiber.StatusInternalServerError, "DB_QUERY_FAILED", err.Error())
	}
	if conn == nil {
		return models.SendError(c, fiber.StatusNotFound, "CONNECTION_NOT_FOUND", "Redis connection not found")
	}
	return c.JSON(fiber.Map{"data": conn})
}

func (h *ConnectionHandler) Create(c *fiber.Ctx) error {
	var input models.ConnectionCreateInput
	if err := c.BodyParser(&input); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid JSON payload")
	}

	conn, err := h.svc.Create(c.UserContext(), &input)
	if err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": conn})
}

func (h *ConnectionHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var input models.ConnectionUpdateInput
	if err := c.BodyParser(&input); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid JSON payload")
	}

	conn, err := h.svc.Update(c.UserContext(), id, &input)
	if err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "UPDATE_FAILED", err.Error())
	}

	return c.JSON(fiber.Map{"data": conn})
}

func (h *ConnectionHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.svc.Delete(c.UserContext(), id); err != nil {
		return models.SendError(c, fiber.StatusInternalServerError, "DELETE_FAILED", err.Error())
	}
	return c.JSON(fiber.Map{"message": "Connection deleted successfully"})
}

func (h *ConnectionHandler) TestByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.svc.TestConnection(c.UserContext(), id); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "REDIS_CONNECTION_FAILED", err.Error())
	}
	return c.JSON(fiber.Map{"status": "ok", "message": "Redis connection successful"})
}

func (h *ConnectionHandler) TestInput(c *fiber.Ctx) error {
	var input models.ConnectionCreateInput
	if err := c.BodyParser(&input); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid JSON payload")
	}

	if err := h.svc.TestInput(c.UserContext(), &input); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "REDIS_CONNECTION_FAILED", err.Error())
	}

	return c.JSON(fiber.Map{"status": "ok", "message": "Redis connection successful"})
}
