package model

import "time"

type Server struct {
	ID                  string
	Status              string
	Ipv4                string
	Port                int
	HealthEndpoint      string
	HealthCheckInterval int //seconds
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
