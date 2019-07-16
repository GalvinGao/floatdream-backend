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

const (
	StatusFetchTimeout = time.Second * 5
)

var (
	ServerUnreachableResponse = ServerStatusResponse{
		Operating: false,
		Latency:   -1,
	}
)

type CachedServerStatus struct {
	ServerAddress  string
	LastLatency    int64
	LastUpdate     time.Time
	UpdateInterval time.Duration
}

func getFreshStatus(serverAddress string) (latency int64) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", serverAddress, StatusFetchTimeout)
	if err != nil {
		return -1
	}
	end := time.Now()
	if err := conn.Close(); err != nil {
		return -1
	}
	return end.Sub(start).Nanoseconds()
}

func NewStatusCache(serverAddress string, updateInterval time.Duration) *CachedServerStatus {
	currentLatency := getFreshStatus(serverAddress)
	return &CachedServerStatus{
		ServerAddress:  serverAddress,
		LastLatency:    currentLatency,
		LastUpdate:     time.Now(),
		UpdateInterval: updateInterval,
	}
}

func (s *CachedServerStatus) Get() (online bool, latency int64) {
	if s.LastUpdate.Add(s.UpdateInterval).Before(time.Now()) {
		s.LastLatency = getFreshStatus(s.ServerAddress)
		s.LastUpdate = time.Now()
	}
	return s.LastLatency != -1, s.LastLatency
}

func provideServerStatus(c echo.Context) error {
	online, latency := ServerStatusCache.Get()
	if !online {
		return c.JSON(http.StatusServiceUnavailable, ServerUnreachableResponse)
	}
	return c.JSON(http.StatusOK, ServerStatusResponse{
		online,
		latency,
	})
}
