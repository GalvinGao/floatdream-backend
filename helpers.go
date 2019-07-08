package main

import (
	"github.com/labstack/echo"
	"strconv"
)

type ErrorMessage struct {
	Message string `json:"message"`
}

func NewErrorResponse(code int, message string) *echo.HTTPError {
	return echo.NewHTTPError(code, ErrorMessage{
		Message: message,
	})
}

func toUint64(s string) uint64 {
	r, _ := strconv.ParseUint(s, 10, 64)
	return r
}

func toInt64(s string) int64 {
	r, _ := strconv.ParseInt(s, 10, 64)
	return r
}
