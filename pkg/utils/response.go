package utils

import "github.com/gofiber/fiber/v2"

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func JSONSuccess(c *fiber.Ctx, statusCode int, message string, data interface{}) error {
	return c.Status(statusCode).JSON(Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func JSONError(c *fiber.Ctx, statusCode int, message string, errDetail string) error {
	return c.Status(statusCode).JSON(Response{
		Success: false,
		Message: message,
		Error:   errDetail,
	})
}
