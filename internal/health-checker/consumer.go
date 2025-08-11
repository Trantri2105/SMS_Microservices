package health_checker

import (
	"VCS_SMS_Microservice/internal/server-service/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Consumer interface {
	Start()
	Stop()
}

type consumer struct {
	kafkaReader  *kafka.Reader
	kafkaWriter  *kafka.Writer
	serverClient ServerClient
	logger       *zap.Logger
}

type serverEvent struct {
	ID                  string `json:"id"`
	Ipv4                string `json:"ipv4"`
	Port                int    `json:"port"`
	HealthEndpoint      string `json:"health_endpoint"`
	HealthCheckInterval int    `json:"health_check_interval"`
}

func (c *consumer) Start() {
	go func() {
		for {
			m, err := c.kafkaReader.FetchMessage(context.Background())
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				err = fmt.Errorf("consumer.Start: %w", err)
				c.logger.Log(zap.ErrorLevel, "failed to fetch message", zap.Error(err))
				continue
			}
			if m.Value == nil {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			var event serverEvent
			if err = json.Unmarshal(m.Value, &event); err != nil {
				err = fmt.Errorf("consumer.Start: %w", err)
				c.logger.Log(zap.ErrorLevel, "failed to unmarshal message", zap.Error(err))
				err = c.kafkaReader.CommitMessages(ctx, m)
				cancel()
				if err != nil {
					err = fmt.Errorf("consumer.Start: %w", err)
					c.logger.Log(zap.ErrorLevel, "failed to commit messages", zap.Error(err))
				}
				continue
			}
			err = c.PerformHealthCheck(ctx, event.ID, event.Ipv4, event.Port, event.HealthCheckInterval, event.HealthEndpoint)
			if err != nil {
				cancel()
				err = fmt.Errorf("consumer.Start: %w", err)
				c.logger.Log(zap.ErrorLevel, "failed to perform health check", zap.Error(err))
				continue
			}
			err = c.kafkaReader.CommitMessages(ctx, m)
			cancel()
			if err != nil {
				err = fmt.Errorf("consumer.Start: %w", err)
				c.logger.Log(zap.ErrorLevel, "failed to commit messages", zap.Error(err))
			}
		}
	}()
}

func (c *consumer) PerformHealthCheck(ctx context.Context, serverID string, ipv4 string, port int, healthCheckInterval int, healthEndpoint string) error {
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
	err = c.kafkaWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(serverID),
		Value: b,
	})
	if err != nil {
		return fmt.Errorf("Checker.PerformHealthCheck: %w", err)
	}
	return nil
}

// Stop Consumer will also close kafka reader but not kafka writer
func (c *consumer) Stop() {
	c.kafkaReader.Close()
}

func NewConsumer(reader *kafka.Reader, writer *kafka.Writer, serverClient ServerClient, logger *zap.Logger) Consumer {
	return &consumer{
		kafkaReader:  reader,
		kafkaWriter:  writer,
		serverClient: serverClient,
		logger:       logger,
	}
}
