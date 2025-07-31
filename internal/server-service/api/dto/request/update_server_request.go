package request

type UpdateServerRequest struct {
	ServerName          string `json:"server_name"`
	Ipv4                string `json:"ipv4" binding:"omitempty,ipv4"`
	Port                *int   `json:"port" binding:"omitempty,gte=1"`
	HealthEndpoint      string `json:"health_endpoint"`
	HealthCheckInterval *int   `json:"health_check_interval" binding:"omitempty,gte=1"`
}
