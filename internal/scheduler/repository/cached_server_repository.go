package repository

import (
	"VCS_SMS_Microservice/internal/scheduler/model"
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type cachedServerRepository struct {
	redis    *redis.Client
	repo     ServerRepository
	cacheTTL time.Duration
}

func (*cachedServerRepository) getServerCachedKey(id string) string {
	return fmt.Sprintf("server:%s", id)
}

func (c *cachedServerRepository) CreateServer(ctx context.Context, server model.Server) (model.Server, error) {
	return c.repo.CreateServer(ctx, server)
}

func (c *cachedServerRepository) GetServerByIds(ctx context.Context, serverIds []string) ([]model.Server, error) {
	keys := make([]string, len(serverIds))
	for i, id := range serverIds {
		keys[i] = c.getServerCachedKey(id)
	}
	res, err := c.redis.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("cachedServerRepository.GetServerByIds: %w", err)
	}
	var servers []model.Server
	var missingIds []string
	for i, data := range res {
		if data == nil {
			missingIds = append(missingIds, serverIds[i])
			continue
		}
		b := []byte(data.(string))
		var server model.Server
		if e := gob.NewDecoder(bytes.NewBuffer(b)).Decode(&server); e != nil {
			return nil, fmt.Errorf("cachedServerRepository.GetServerByIds: %w", err)
		}
		servers = append(servers, server)
	}
	if len(missingIds) > 0 {
		remainServers, e := c.repo.GetServerByIds(ctx, serverIds)
		if e != nil {
			return nil, fmt.Errorf("cachedServerRepository.GetServerByIds: %w", e)
		}
		servers = append(servers, remainServers...)
		for _, server := range remainServers {
			c.redis.Set(ctx, c.getServerCachedKey(server.ID), server, c.cacheTTL)
		}
	}
	return servers, nil
}

func (c *cachedServerRepository) UpdateServer(ctx context.Context, updatedData model.Server) (model.Server, error) {
	err := c.redis.Del(ctx, c.getServerCachedKey(updatedData.ID)).Err()
	if err != nil {
		return model.Server{}, fmt.Errorf("cachedServerRepository.UpdateServer: %w", err)
	}
	return c.repo.UpdateServer(ctx, updatedData)
}

func (c *cachedServerRepository) DeleteServerById(ctx context.Context, serverId string) error {
	err := c.redis.Del(ctx, c.getServerCachedKey(serverId)).Err()
	if err != nil {
		return fmt.Errorf("cachedServerRepository.DeleteServer: %w", err)
	}
	return c.repo.DeleteServerById(ctx, serverId)
}

func NewCachedServerRepository(redis *redis.Client, repo ServerRepository, cacheTTL time.Duration) ServerRepository {
	return &cachedServerRepository{
		redis: redis,
		repo:  repo,
	}
}
