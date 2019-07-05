package floatdream_backend_copy

import "github.com/labstack/echo"

type ErrorMessage struct {
	Message string
}

func NewErrorResponse(code int, message string) *echo.HTTPError {
	return echo.NewHTTPError(code, ErrorMessage{
		Message: message,
	})
}
