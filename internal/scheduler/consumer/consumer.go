package consumer

import (
	"VCS_SMS_Microservice/internal/scheduler/model"
	"VCS_SMS_Microservice/internal/scheduler/repository"
	"VCS_SMS_Microservice/pkg/infra"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"go.uber.org/zap"
)

type ServerConsumer interface {
	Start()
	Stop()
}

type serverConsumer struct {
	repo   repository.ServerRepository
	kafka  infra.KafkaReader
	logger *zap.Logger
}

type serverEvent struct {
	Payload struct {
		Op     string `json:"op"`
		Before struct {
			Id string `json:"id"`
		} `json:"before"`
		After struct {
			Id                  string `json:"id"`
			Ipv4                string `json:"ipv4"`
			Port                int    `json:"port"`
			HealthCheckInterval int    `json:"health_check_interval"`
			HealthEndpoint      string `json:"health_endpoint"`
		} `json:"after"`
	} `json:"payload"`
}

func (s *serverConsumer) Start() {
	go func() {
		for {
			m, err := s.kafka.FetchMessage(context.Background())
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				err = fmt.Errorf("ServerConsumer.Start: %w", err)
				s.logger.Log(zap.ErrorLevel, "failed to fetch message", zap.Error(err))
				continue
			}
			if m.Value == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err = s.kafka.CommitMessages(ctx, m)
				cancel()
				if err != nil {
					err = fmt.Errorf("serverConsumer.Start: %w", err)
					s.logger.Log(zap.ErrorLevel, "failed to commit messages", zap.Error(err))
				}
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			var event serverEvent
			if err = json.Unmarshal(m.Value, &event); err != nil {
				err = fmt.Errorf("ServerConsumer.Start: %w", err)
				s.logger.Log(zap.ErrorLevel, "failed to unmarshal message", zap.Error(err))
				err = s.kafka.CommitMessages(ctx, m)
				cancel()
				if err != nil {
					err = fmt.Errorf("consumer.Start: %w", err)
					s.logger.Log(zap.ErrorLevel, "failed to commit messages", zap.Error(err))
				}
				continue
			}
			switch event.Payload.Op {
			case "c":
				_, err = s.repo.CreateServer(ctx, model.Server{
					ID:                  event.Payload.After.Id,
					Ipv4:                event.Payload.After.Ipv4,
					Port:                event.Payload.After.Port,
					HealthEndpoint:      event.Payload.After.HealthEndpoint,
					HealthCheckInterval: event.Payload.After.HealthCheckInterval,
					NextHealthCheckAt:   time.Now().Add(1 * time.Second),
				})
				if err != nil {
					cancel()
					err = fmt.Errorf("ServerConsumer.Start: %w", err)
					s.logger.Log(zap.ErrorLevel, "failed to create server", zap.Error(err))
					continue
				}
			case "u":
				_, err = s.repo.UpdateServer(ctx, model.Server{
					ID:                  event.Payload.After.Id,
					Ipv4:                event.Payload.After.Ipv4,
					Port:                event.Payload.After.Port,
					HealthEndpoint:      event.Payload.After.HealthEndpoint,
					HealthCheckInterval: event.Payload.After.HealthCheckInterval,
				})
				if err != nil {
					cancel()
					err = fmt.Errorf("ServerConsumer.Start: %w", err)
					s.logger.Log(zap.ErrorLevel, "failed to update server", zap.Error(err))
					continue
				}
			case "d":
				err = s.repo.DeleteServerById(ctx, event.Payload.Before.Id)
				if err != nil {
					cancel()
					err = fmt.Errorf("ServerConsumer.Start: %w", err)
					s.logger.Log(zap.ErrorLevel, "failed to delete server", zap.Error(err))
					continue
				}
			default:
				s.logger.Log(zap.InfoLevel, "unknown event", zap.String("event", event.Payload.Op))
			}
			err = s.kafka.CommitMessages(ctx, m)
			cancel()
			if err != nil {
				err = fmt.Errorf("ServerConsumer.Start: %w", err)
				s.logger.Log(zap.ErrorLevel, "failed to commit messages", zap.Error(err))
			}
		}
	}()
}

func (s *serverConsumer) Stop() {
	s.kafka.Close()
}

func NewServerConsumer(repo repository.ServerRepository, logger *zap.Logger, kafka infra.KafkaReader) ServerConsumer {
	return &serverConsumer{
		repo:   repo,
		kafka:  kafka,
		logger: logger,
	}
}
