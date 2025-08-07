package health_checker

import (
	"VCS_SMS_Microservice/internal/server-service/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

type Checker interface {
	PerformHealthCheck(ctx context.Context, serverID string, ipv4 string, port int, healthCheckInterval int, healthEndpoint string) error
}

type checker struct {
	serverClient ServerClient
	kafka        *kafka.Writer
}

func (c *checker) PerformHealthCheck(ctx context.Context, serverID string, ipv4 string, port int, healthCheckInterval int, healthEndpoint string) error {
	start := time.Now()
	res, err := c.serverClient.GetServerHealthCheck(ctx, ipv4, port, healthEndpoint)
	if err != nil {
		return fmt.Errorf("Checker.PerformHealthCheck: %w", err)
	}
	healthCheck := struct {
		ServerID                       string    `json:"server_id"`
		Status                         string    `json:"status"`
		StatusNumeric                  int       `json:"status_numeric"`
		Timestamp                      time.Time `json:"timestamp"`
		Attempts                       int       `json:"attempts"`
		IntervalSinceLastHealthCheckMs int64     `json:"interval_since_last_health_check_ms"`
	}{
		ServerID:  serverID,
		Timestamp: res.Timestamp,
		Attempts:  res.Attempts,
	}
	if res.Error != nil {
		if errors.Is(res.Error, syscall.ECONNREFUSED) {
			healthCheck.Status = model.ServerStatusInactive
		} else {
			healthCheck.Status = model.ServerStatusNetworkError
		}
	} else {
		if res.StatusCode >= 200 && res.StatusCode < 300 {
			healthCheck.Status = model.ServerStatusHealthy
		} else if res.StatusCode >= 400 && res.StatusCode < 500 {
			healthCheck.Status = model.ServerStatusConfigurationError
		} else {
			healthCheck.Status = model.ServerStatusUnhealthy
		}
	}
	if healthCheck.Status == model.ServerStatusHealthy {
		healthCheck.StatusNumeric = 1
	}
	healthCheck.IntervalSinceLastHealthCheckMs = int64(healthCheckInterval*1000) + healthCheck.Timestamp.Sub(start).Milliseconds()
	b, err := json.Marshal(healthCheck)
	if err != nil {
		return fmt.Errorf("Checker.PerformHealthCheck: %w", err)
	}
	err = c.kafka.WriteMessages(ctx, kafka.Message{
		Key:   []byte(serverID),
		Value: b,
	})
	if err != nil {
		return fmt.Errorf("Checker.PerformHealthCheck: %w", err)
	}
	return nil
}

func NewChecker(serverClient ServerClient, kafka *kafka.Writer) Checker {
	return &checker{
		serverClient: serverClient,
		kafka:        kafka,
	}
}
