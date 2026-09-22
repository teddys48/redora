package models

import "github.com/gofiber/fiber/v2"

type APIErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIError struct {
	Error APIErrorDetail `json:"error"`
}

func NewAPIError(code, message string) APIError {
	return APIError{
		Error: APIErrorDetail{
			Code:    code,
			Message: message,
		},
	}
}

func SendError(c *fiber.Ctx, statusCode int, code, message string) error {
	return c.Status(statusCode).JSON(NewAPIError(code, message))
}
