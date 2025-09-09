package health_check_consumer

import (
	"VCS_SMS_Microservice/internal/server-service/model"
	"VCS_SMS_Microservice/internal/server-service/repository"
	"VCS_SMS_Microservice/pkg/infra"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"go.uber.org/zap"
)

type HealthCheckConsumer interface {
	Start()
	Stop()
}

type healthCheckConsumer struct {
	kafkaReader infra.KafkaReader
	serverRepo  repository.ServerRepository
	logger      *zap.Logger
}

type healthCheckEvent struct {
	ServerID string `json:"server_id"`
	Status   string `json:"status"`
}

func (h *healthCheckConsumer) Start() {
	go func() {
		for {
			m, err := h.kafkaReader.FetchMessage(context.Background())
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				err = fmt.Errorf("healthCheckConsumer.Start: %w", err)
				h.logger.Log(zap.ErrorLevel, "failed to fetch message", zap.Error(err))
				continue
			}
			if m.Value == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err = h.kafkaReader.CommitMessages(ctx, m)
				cancel()
				if err != nil {
					err = fmt.Errorf("healthCheckConsumer.Start: %w", err)
					h.logger.Log(zap.ErrorLevel, "failed to commit messages", zap.Error(err))
				}
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			var event healthCheckEvent
			if err = json.Unmarshal(m.Value, &event); err != nil {
				err = fmt.Errorf("healthCheckConsumer.Start: %w", err)
				h.logger.Log(zap.ErrorLevel, "failed to unmarshal message", zap.Error(err))
				err = h.kafkaReader.CommitMessages(ctx, m)
				cancel()
				if err != nil {
					err = fmt.Errorf("healthCheckConsumer.Start: %w", err)
					h.logger.Log(zap.ErrorLevel, "failed to commit messages", zap.Error(err))
				}
				continue
			}
			_, err = h.serverRepo.UpdateServer(ctx, model.Server{
				ID:     event.ServerID,
				Status: event.Status,
			})
			if err != nil {
				cancel()
				err = fmt.Errorf("healthCheckConsumer.Start: %w", err)
				h.logger.Log(zap.ErrorLevel, "failed to update server", zap.Error(err))
				continue
			}
			err = h.kafkaReader.CommitMessages(ctx, m)
			cancel()
			if err != nil {
				err = fmt.Errorf("healthCheckConsumer.Start: %w", err)
				h.logger.Log(zap.ErrorLevel, "failed to commit messages", zap.Error(err))
			}
		}
	}()
}

func (h *healthCheckConsumer) Stop() {
	h.kafkaReader.Close()
}

func NewHealthCheckConsumer(reader infra.KafkaReader, serverRepo repository.ServerRepository, logger *zap.Logger) HealthCheckConsumer {
	return &healthCheckConsumer{
		kafkaReader: reader,
		serverRepo:  serverRepo,
		logger:      logger,
	}
}
