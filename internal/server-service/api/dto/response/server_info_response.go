package response

import "time"

type ServerInfoResponse struct {
	ID                  string    `json:"id"`
	ServerName          string    `json:"server_name"`
	Status              string    `json:"status"`
	Ipv4                string    `json:"ipv4"`
	Port                int       `json:"port"`
	HealthEndpoint      string    `json:"health_endpoint"`
	HealthCheckInterval int       `json:"health_check_interval"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
