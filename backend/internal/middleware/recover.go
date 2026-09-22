package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/redora/redora/backend/internal/models"
)

func Recover() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				var errStr string
				if ok {
					errStr = err.Error()
				} else {
					errStr = fmt.Sprintf("%v", r)
				}

				slog.Error("Unhandled panic recovered",
					"error", errStr,
					"stack", string(debug.Stack()),
					"path", c.Path(),
				)

				_ = models.SendError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An internal server error occurred")
			}
		}()
		return c.Next()
	}
}
