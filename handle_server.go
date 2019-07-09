package main

import (
	"github.com/labstack/echo"
	"net"
	"net/http"
	"time"
)

type ServerStatusResponse struct {
	Operating bool  `json:"operating"`
	Latency   int64 `json:"latency"`
}

var (
	ServerUnreachableResponse = ServerStatusResponse{
		Operating: false,
		Latency:   -1,
	}
)

func fetchServerStatus(serverAddress string) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", serverAddress, time.Second*5)
		if err != nil {
			return c.JSON(http.StatusServiceUnavailable, ServerUnreachableResponse)
		}
		end := time.Now()
		if err := conn.Close(); err != nil {
			return c.JSON(http.StatusServiceUnavailable, ServerUnreachableResponse)
		}
		return c.JSON(http.StatusOK, ServerStatusResponse{
			Operating: true,
			Latency:   end.Sub(start).Nanoseconds(),
		})
	}
}
