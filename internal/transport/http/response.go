package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func SuccessResponse(c echo.Context, data interface{}, message string) error {
	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

func ErrorResponse(c echo.Context, statusCode int, message string) error {
	return c.JSON(statusCode, Response{
		Success: false,
		Data:    nil,
		Message: message,
	})
}
