package repository

import (
	apperrors "VCS_SMS_Microservice/internal/scheduler/errors"
	"VCS_SMS_Microservice/internal/scheduler/model"
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ServerRepository interface {
	CreateServer(ctx context.Context, server model.Server) (model.Server, error)
	GetServersForHealthCheck(ctx context.Context) ([]model.Server, error)
	UpdateServer(ctx context.Context, updatedData model.Server) (model.Server, error)
	DeleteServerById(ctx context.Context, serverId string) error
	UpdateServersNextHealthCheckByIds(ctx context.Context, serverIds []string) error
}

type serverRepository struct {
	db *gorm.DB
}

func (s *serverRepository) CreateServer(ctx context.Context, server model.Server) (model.Server, error) {
	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&server)
	if result.Error != nil {
		return server, fmt.Errorf("ServerRepository.CreateServer: %w", result.Error)
	}
	return server, nil
}

func (s *serverRepository) GetServersForHealthCheck(ctx context.Context) ([]model.Server, error) {
	var servers []model.Server
	result := s.db.WithContext(ctx).Where("next_health_check_at <= NOW()").Find(&servers)
	if result.Error != nil {
		return servers, fmt.Errorf("ServerRepository.GetServerById: %w", result.Error)
	}
	return servers, nil
}

func (s *serverRepository) UpdateServer(ctx context.Context, updatedData model.Server) (model.Server, error) {
	var server model.Server
	result := s.db.WithContext(ctx).Model(&server).Clauses(clause.Returning{}).Where("id = ?", updatedData.ID).Updates(updatedData)
	if result.Error != nil {
		return server, fmt.Errorf("UserRepository.UpdateServer: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return server, fmt.Errorf("ServerRepository.UpdateServer: %w", apperrors.ErrServerNotFound)
	}
	return server, nil
}

func (s *serverRepository) UpdateServersNextHealthCheckByIds(ctx context.Context, serverIds []string) error {
	res := s.db.WithContext(ctx).Model(&model.Server{}).Where("id IN ?", serverIds).Update("next_health_check_at", gorm.Expr("NOW() + (health_check_interval * INTERVAL '1 second')"))
	if res.Error != nil {
		return fmt.Errorf("ServerRepository.UpdateServersNextHealthCheckByIds: %w", res.Error)
	}
	return nil
}

func (s *serverRepository) DeleteServerById(ctx context.Context, serverId string) error {
	result := s.db.WithContext(ctx).Where("id = ?", serverId).Delete(&model.Server{})
	if result.Error != nil {
		return fmt.Errorf("ServerRepository.DeleteServerById: %w", result.Error)
	}
	return nil
}

func NewServerRepository(db *gorm.DB) ServerRepository {
	return &serverRepository{
		db: db,
	}
}
