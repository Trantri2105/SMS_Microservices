package model

import "time"

const (
	ServerStatusHealthy            = "healthy"
	ServerStatusUnhealthy          = "unhealthy"
	ServerStatusPending            = "pending"
	ServerStatusInactive           = "inactive"
	ServerStatusConfigurationError = "configuration_error"
	ServerStatusNetworkError       = "network_error"
)

type Server struct {
	ID                  string `gorm:"default:(-)"`
	ServerName          string
	Status              string
	Ipv4                string
	Port                int
	HealthEndpoint      string
	HealthCheckInterval int //seconds
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
