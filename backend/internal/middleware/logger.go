package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"
)

var passwordRegex = regexp.MustCompile(`(?i)"password"\s*:\s*"[^"]*"`)

func sanitizeBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	// Limit logged body length to 2KB to prevent memory issues with large payloads
	// if len(body) > 2048 {
	// 	body = append(body[:2048], []byte("...[TRUNCATED]")...)
	// }
	// Mask passwords in request/response bodies
	sanitized := passwordRegex.ReplaceAllString(string(body), `"password":"***"`)

	// Compact JSON formatting if valid JSON
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(sanitized)); err == nil {
		return compact.String()
	}
	return sanitized
}

func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		reqBody := sanitizeBody(c.Body())

		err := c.Next()
		duration := time.Since(start)

		status := c.Response().StatusCode()
		method := c.Method()
		path := c.Path()
		ip := c.IP()
		respBody := sanitizeBody(c.Response().Body())

		if err != nil {
			slog.Error("HTTP API Request Completed with Error",
				"status", status,
				"method", method,
				"path", path,
				"duration", duration.String(),
				"ip", ip,
				"request_body", reqBody,
				"response_body", respBody,
				"error", err,
			)
		} else {
			slog.Info("HTTP API Request Completed",
				"status", status,
				"method", method,
				"path", path,
				"duration", duration.String(),
				"ip", ip,
				"request_body", reqBody,
				"response_body", respBody,
			)
		}

		return err
	}
}
