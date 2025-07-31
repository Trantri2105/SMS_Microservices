package model

import "time"

type HealthCheck struct {
	ServerID                       string    `json:"server_id"`
	Status                         string    `json:"status"`
	StatusNumeric                  int       `json:"status_numeric"` // 1 for healthy, 0 for unhealthy and unavailable status
	Timestamp                      time.Time `json:"timestamp"`
	LatencyMs                      int64     `json:"latency_ms"`
	Attempts                       int       `json:"attempts"`
	IntervalSinceLastHealthCheckMs int64     `json:"interval_since_last_health_check_ms"`
}
