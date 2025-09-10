package scheduler

import (
	"VCS_SMS_Microservice/internal/scheduler/repository"
	"VCS_SMS_Microservice/pkg/infra"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type ServerScheduler interface {
	Start()
	Stop()
}

type serverScheduler struct {
	ticker     *time.Ticker
	logger     *zap.Logger
	stopChan   chan struct{}
	serverRepo repository.ServerRepository
	kafka      infra.KafkaWriter
}

func (s *serverScheduler) Start() {
	go func() {
		s.ticker = time.NewTicker(1 * time.Second)
		for {
			select {
			case <-s.ticker.C:
				s.onTick()
			case <-s.stopChan:
				s.kafka.Close()
				return
			}
		}
	}()
}

func (s *serverScheduler) Stop() {
	s.stopChan <- struct{}{}
}

func (s *serverScheduler) onTick() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	servers, err := s.serverRepo.GetServersForHealthCheck(ctx)
	if err != nil {
		s.logger.Error("failed to fetch servers", zap.Error(fmt.Errorf("serverScheduler.onTick: %w", err)))
		return
	}
	if len(servers) > 0 {
		var messages []kafka.Message
		var ids []string
		for _, server := range servers {
			b, e := json.Marshal(server)
			if e != nil {
				e = fmt.Errorf("serverScheduler.onTick: %w", e)
				s.logger.Error("failed to marshal server info", zap.Error(e), zap.String("server_id", server.ID))
			} else {
				messages = append(messages, kafka.Message{
					Key:   []byte(server.ID),
					Value: b,
				})
				ids = append(ids, server.ID)
			}
		}
		err = s.kafka.WriteMessages(ctx, messages...)
		if err != nil {
			s.logger.Error("failed to write messages to kafka", zap.Error(fmt.Errorf("serverScheduler.onTick: %w", err)))
			return
		}
		err = s.serverRepo.UpdateServersNextHealthCheckByIds(ctx, ids)
		if err != nil {
			s.logger.Error("failed to update servers next health check", zap.Error(fmt.Errorf("serverScheduler.onTick: %w", err)))
		}
	}
}

func NewServerScheduler(logger *zap.Logger, serverRepository repository.ServerRepository, kafka infra.KafkaWriter) ServerScheduler {
	return &serverScheduler{
		logger:     logger,
		stopChan:   make(chan struct{}),
		serverRepo: serverRepository,
		kafka:      kafka,
	}
}
