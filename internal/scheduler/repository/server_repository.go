package repository

import (
	apperrors "VCS_SMS_Microservice/internal/scheduler/errors"
	"VCS_SMS_Microservice/internal/scheduler/model"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ServerRepository interface {
	CreateServer(ctx context.Context, server model.Server) (model.Server, error)
	GetMultipleServersByIds(ctx context.Context, serverIds []string) ([]model.Server, error)
	UpdateServer(ctx context.Context, updatedData model.Server) (model.Server, error)
	DeleteServerById(ctx context.Context, serverId string) error
	GetAllServers(ctx context.Context) ([]model.Server, error)
}

type serverRepository struct {
	db *gorm.DB
}

func (s *serverRepository) GetAllServers(ctx context.Context) ([]model.Server, error) {
	var servers []model.Server
	res := s.db.WithContext(ctx).Find(&servers)
	if res.Error != nil {
		return nil, fmt.Errorf("ServerRepository.GetAllServers: %w", res.Error)
	}
	return servers, nil
}

func (s *serverRepository) CreateServer(ctx context.Context, server model.Server) (model.Server, error) {
	result := s.db.WithContext(ctx).Create(&server)
	if result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "servers_server_name_key" {
				return server, fmt.Errorf("ServerRepository.CreateServer: %w", apperrors.ErrServerNameAlreadyExists)
			}
		}
		return server, fmt.Errorf("ServerRepository.CreateServer: %w", result.Error)
	}
	return server, nil
}

func (s *serverRepository) GetMultipleServersByIds(ctx context.Context, serverIds []string) ([]model.Server, error) {
	var servers []model.Server
	result := s.db.WithContext(ctx).Where("id IN ?", serverIds).Find(&servers)
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
