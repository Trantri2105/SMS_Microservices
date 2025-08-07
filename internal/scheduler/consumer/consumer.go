package consumer

import (
	"VCS_SMS_Microservice/internal/scheduler/model"
	"VCS_SMS_Microservice/internal/scheduler/repository"
	"VCS_SMS_Microservice/internal/scheduler/scheduler"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type ServerConsumer interface {
	Start()
	Stop()
}

type serverConsumer struct {
	repo      repository.ServerRepository
	timewheel scheduler.TimeWheel
	kafka     *kafka.Reader
	logger    *zap.Logger
	stopChan  chan struct{}
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
				err = fmt.Errorf("ServerConsumer.Start: %w", err)
				s.logger.Log(zap.ErrorLevel, "failed to fetch message", zap.Error(err))
				if errors.Is(err, io.EOF) {
					return
				}
				continue
			}
			if m.Value == nil {
				continue
			}
			var event serverEvent
			if e := json.Unmarshal(m.Value, &event); e != nil {
				err = fmt.Errorf("ServerConsumer.Start: %w", e)
				s.logger.Log(zap.ErrorLevel, "failed to unmarshal message", zap.Error(err))
				continue
			}
			switch event.Payload.Op {
			case "c":
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				createdServer, e := s.repo.CreateServer(ctx, model.Server{
					ID:                  event.Payload.After.Id,
					Ipv4:                event.Payload.After.Ipv4,
					Port:                event.Payload.After.Port,
					HealthEndpoint:      event.Payload.After.HealthEndpoint,
					HealthCheckInterval: event.Payload.After.HealthCheckInterval,
				})
				cancel()
				if e != nil {
					err = fmt.Errorf("ServerConsumer.Start: %w", e)
					s.logger.Log(zap.ErrorLevel, "failed to create server", zap.Error(err))
					continue
				}
				s.timewheel.AddServer(createdServer)
			case "u":
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, e := s.repo.UpdateServer(ctx, model.Server{
					ID:                  event.Payload.After.Id,
					Ipv4:                event.Payload.After.Ipv4,
					Port:                event.Payload.After.Port,
					HealthEndpoint:      event.Payload.After.HealthEndpoint,
					HealthCheckInterval: event.Payload.After.HealthCheckInterval,
				})
				cancel()
				if e != nil {
					err = fmt.Errorf("ServerConsumer.Start: %w", e)
					s.logger.Log(zap.ErrorLevel, "failed to update server", zap.Error(err))
					continue
				}
			case "d":
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				e := s.repo.DeleteServerById(ctx, event.Payload.Before.Id)
				cancel()
				if e != nil {
					err = fmt.Errorf("ServerConsumer.Start: %w", e)
					s.logger.Log(zap.ErrorLevel, "failed to delete server", zap.Error(err))
					continue
				}
			default:
				s.logger.Log(zap.InfoLevel, "unknown event", zap.String("event", event.Payload.Op))
			}
			err = s.kafka.CommitMessages(context.Background(), m)
			if err != nil {
				err = fmt.Errorf("ServerConsumer.Start: %w", err)
				s.logger.Log(zap.ErrorLevel, "failed to commit message", zap.Error(err))
			}
		}
	}()
}

func (s *serverConsumer) Stop() {
	s.kafka.Close()
}

func NewServerConsumer(repo repository.ServerRepository, tw scheduler.TimeWheel, logger *zap.Logger, kafka *kafka.Reader) ServerConsumer {
	return &serverConsumer{
		repo:      repo,
		timewheel: tw,
		kafka:     kafka,
		logger:    logger,
	}
}
