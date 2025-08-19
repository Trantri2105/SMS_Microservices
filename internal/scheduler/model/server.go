package model

import "time"

type Server struct {
	ID                  string    `json:"id"`
	Ipv4                string    `json:"ipv4"`
	Port                int       `json:"port"`
	HealthEndpoint      string    `json:"health_endpoint"`
	HealthCheckInterval int       `json:"health_check_interval"` // second
	NextHealthCheckAt   time.Time `json:"next_health_check_at"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
