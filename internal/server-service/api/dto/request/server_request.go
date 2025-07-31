package request

type ServerRequest struct {
	ServerName          string `json:"server_name" binding:"required" validate:"required"`
	Ipv4                string `json:"ipv4" binding:"required,ipv4" validate:"required,ipv4"`
	Port                *int   `json:"port" binding:"required,gte=1" validate:"required,gte=1"`
	HealthEndpoint      string `json:"health_endpoint" binding:"required" validate:"required"`
	HealthCheckInterval *int   `json:"health_check_interval" binding:"required,gte=1" validate:"required,gte=1"`
}
