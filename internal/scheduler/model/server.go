package model

import "time"

type Server struct {
	ID                  string
	Ipv4                string
	Port                int
	HealthEndpoint      string
	HealthCheckInterval int //seconds
	NextHealthCheckAt   time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
