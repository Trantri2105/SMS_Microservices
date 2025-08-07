package scheduler

import (
	"VCS_SMS_Microservice/internal/scheduler/model"
	"VCS_SMS_Microservice/internal/scheduler/repository"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type TimeWheel interface {
	Start()
	Stop()
	AddServer(server model.Server)
}

type timeWheel struct {
	ticker        *time.Ticker
	slots         [][]task
	slotCount     int
	currentSlot   int
	addServerChan chan model.Server
	workerCount   int
	jobQueue      chan []model.Server
	wg            sync.WaitGroup
	stopChan      chan struct{}
	logger        *zap.Logger
	kafka         *kafka.Writer
	repo          repository.ServerRepository
}

type task struct {
	server model.Server
	slot   int
	laps   int
}

func (tw *timeWheel) Start() {
	defer tw.ticker.Stop()
	tw.startWorkerPool()
	for {
		select {
		case <-tw.ticker.C:
			tw.onTick()
		case server := <-tw.addServerChan:
			tw.addTask(server)
		case <-tw.stopChan:
			close(tw.jobQueue)
			return
		}
	}
}

func (tw *timeWheel) Stop() {
	close(tw.stopChan)
	tw.wg.Wait()
	tw.kafka.Close()
}

func (tw *timeWheel) AddServer(server model.Server) {
	tw.addServerChan <- server
}

func (tw *timeWheel) addTask(server model.Server) {
	laps := server.HealthCheckInterval / tw.slotCount
	if server.HealthCheckInterval%tw.slotCount == 0 {
		laps -= 1
	}
	slot := (tw.currentSlot + server.HealthCheckInterval) % tw.slotCount
	t := task{
		server: server,
		laps:   laps,
		slot:   slot,
	}
	tw.slots[t.slot] = append(tw.slots[t.slot], t)
}

func (tw *timeWheel) startWorkerPool() {
	tw.wg.Add(tw.workerCount)
	for i := 0; i < tw.workerCount; i++ {
		go tw.worker()
	}
}

func (tw *timeWheel) worker() {
	for s := range tw.jobQueue {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		ids := make([]string, len(s))
		for i, server := range s {
			ids[i] = server.ID
		}
		servers, err := tw.repo.GetMultipleServersByIds(ctx, ids)
		if err != nil {
			err = fmt.Errorf("TimeWheel.worker: %w", err)
			tw.logger.Error("failed to get latest servers info", zap.Error(err))
			for _, server := range s {
				tw.AddServer(server)
			}
		} else {
			var messages []kafka.Message
			for _, server := range servers {
				b, err := json.Marshal(server)
				if err != nil {
					err = fmt.Errorf("TimeWheel.worker: %w", err)
					tw.logger.Error("failed to marshal server info", zap.Error(err), zap.String("server_id", server.ID))
				} else {
					messages = append(messages, kafka.Message{
						Key:   []byte(server.ID),
						Value: b,
					})
				}
			}
			err = tw.kafka.WriteMessages(ctx, messages...)
			if err != nil {
				err = fmt.Errorf("TimeWheel.worker: %w", err)
				tw.logger.Error("failed to write messages", zap.Error(err))
			}
			for _, server := range servers {
				tw.AddServer(server)
			}
		}
		cancel()
	}
	tw.wg.Done()
}

func (tw *timeWheel) onTick() {
	tw.currentSlot = (tw.currentSlot + 1) % tw.slotCount
	currentTasks := tw.slots[tw.currentSlot]
	tw.slots[tw.currentSlot] = nil
	var servers []model.Server
	for _, t := range currentTasks {
		if t.laps > 0 {
			t.laps -= 1
			tw.slots[t.slot] = append(tw.slots[t.slot], t)
		} else {
			servers = append(servers, t.server)
		}
	}
	tw.jobQueue <- servers
}

func NewTimeWheel(slotCount int, queueSize int, workerCount int, logger *zap.Logger, repo repository.ServerRepository, kafka *kafka.Writer) TimeWheel {
	return &timeWheel{
		ticker:        time.NewTicker(1 * time.Second),
		slots:         make([][]task, slotCount),
		slotCount:     slotCount,
		currentSlot:   0,
		addServerChan: make(chan model.Server, queueSize),
		jobQueue:      make(chan []model.Server, queueSize),
		workerCount:   workerCount,
		wg:            sync.WaitGroup{},
		stopChan:      make(chan struct{}),
		logger:        logger,
		repo:          repo,
		kafka:         kafka,
	}
}
