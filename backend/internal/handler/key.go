package handler

import (
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/redora/redora/backend/internal/models"
	"github.com/redora/redora/backend/internal/service"
)

type KeyHandler struct {
	svc service.RedisService
}

func NewKeyHandler(svc service.RedisService) *KeyHandler {
	return &KeyHandler{svc: svc}
}

func (h *KeyHandler) RegisterRoutes(router fiber.Router) {
	group := router.Group("/connections/:id/keys")
	group.Get("/", h.ScanKeys)
	group.Post("/", h.CreateKey)
	group.Get("/:key", h.GetKeyDetail)
	group.Put("/:key", h.UpdateKey)
	group.Delete("/:key", h.DeleteKey)
	group.Post("/:key/rename", h.RenameKey)
	group.Post("/:key/expire", h.SetTTL)
}

func (h *KeyHandler) ScanKeys(c *fiber.Ctx) error {
	connID := c.Params("id")
	pattern := c.Query("pattern", "*")
	typeFilter := c.Query("type", "")

	cursorStr := c.Query("cursor", "0")
	cursor, err := strconv.ParseUint(cursorStr, 10, 64)
	if err != nil {
		cursor = 0
	}

	limitStr := c.Query("limit", "100")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		limit = 100
	}

	res, err := h.svc.ScanKeys(c.UserContext(), connID, pattern, typeFilter, cursor, limit)
	if err != nil {
		return models.SendError(c, fiber.StatusInternalServerError, "REDIS_SCAN_FAILED", err.Error())
	}

	return c.JSON(res)
}

func (h *KeyHandler) GetKeyDetail(c *fiber.Ctx) error {
	connID := c.Params("id")
	keyParam := c.Params("key")
	key, err := url.PathUnescape(keyParam)
	if err != nil {
		key = keyParam
	}

	res, err := h.svc.GetKeyDetail(c.UserContext(), connID, key)
	if err != nil {
		return models.SendError(c, fiber.StatusNotFound, "KEY_NOT_FOUND", err.Error())
	}

	return c.JSON(fiber.Map{"data": res})
}

func (h *KeyHandler) CreateKey(c *fiber.Ctx) error {
	connID := c.Params("id")
	var input models.CreateKeyInput
	if err := c.BodyParser(&input); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid JSON payload")
	}

	if err := h.svc.CreateKey(c.UserContext(), connID, &input); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "CREATE_KEY_FAILED", err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Key created successfully"})
}

func (h *KeyHandler) UpdateKey(c *fiber.Ctx) error {
	connID := c.Params("id")
	keyParam := c.Params("key")
	key, err := url.PathUnescape(keyParam)
	if err != nil {
		key = keyParam
	}

	var input models.CreateKeyInput
	if err := c.BodyParser(&input); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid JSON payload")
	}

	if err := h.svc.UpdateKey(c.UserContext(), connID, key, &input); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "UPDATE_KEY_FAILED", err.Error())
	}

	return c.JSON(fiber.Map{"message": "Key updated successfully"})
}

func (h *KeyHandler) DeleteKey(c *fiber.Ctx) error {
	connID := c.Params("id")
	keyParam := c.Params("key")
	key, err := url.PathUnescape(keyParam)
	if err != nil {
		key = keyParam
	}

	if err := h.svc.DeleteKey(c.UserContext(), connID, key); err != nil {
		return models.SendError(c, fiber.StatusInternalServerError, "DELETE_KEY_FAILED", err.Error())
	}

	return c.JSON(fiber.Map{"message": "Key deleted successfully"})
}

func (h *KeyHandler) RenameKey(c *fiber.Ctx) error {
	connID := c.Params("id")
	keyParam := c.Params("key")
	key, err := url.PathUnescape(keyParam)
	if err != nil {
		key = keyParam
	}

	var input models.RenameKeyInput
	if err := c.BodyParser(&input); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid JSON payload")
	}

	if err := h.svc.RenameKey(c.UserContext(), connID, key, input.NewKey); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "RENAME_KEY_FAILED", err.Error())
	}

	return c.JSON(fiber.Map{"message": "Key renamed successfully"})
}

func (h *KeyHandler) SetTTL(c *fiber.Ctx) error {
	connID := c.Params("id")
	keyParam := c.Params("key")
	key, err := url.PathUnescape(keyParam)
	if err != nil {
		key = keyParam
	}

	var input models.SetTTLInput
	if err := c.BodyParser(&input); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid JSON payload")
	}

	if err := h.svc.SetTTL(c.UserContext(), connID, key, input.TTL); err != nil {
		return models.SendError(c, fiber.StatusBadRequest, "SET_TTL_FAILED", err.Error())
	}

	return c.JSON(fiber.Map{"message": "TTL updated successfully"})
}
