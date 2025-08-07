package repository

import (
	apperrors "VCS_SMS_Microservice/internal/server-service/errors"
	"VCS_SMS_Microservice/internal/server-service/model"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ServerRepository interface {
	CreateServer(ctx context.Context, server model.Server) (model.Server, error)
	ImportServers(ctx context.Context, servers []model.Server) (insertedServers []model.Server, nonInsertedServers []model.Server, err error)
	GetServerById(ctx context.Context, serverId string) (model.Server, error)
	UpdateServer(ctx context.Context, updatedData model.Server) (model.Server, error)
	DeleteServerById(ctx context.Context, serverId string) error
	GetServers(ctx context.Context, serverName string, status string, sortBy string, sortOrder string, limit int, offset int) ([]model.Server, error)
}

type serverRepository struct {
	db *gorm.DB
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

func (s *serverRepository) GetServers(ctx context.Context, serverName string, status string, sortBy string, sortOrder string, limit int, offset int) ([]model.Server, error) {
	query := s.db.WithContext(ctx)
	if serverName != "" {
		query = query.Where("server_name LIKE ?", serverName+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).Limit(limit).Offset(offset)
	var servers []model.Server
	result := query.Find(&servers)
	if result.Error != nil {
		return nil, fmt.Errorf("ServerRepository.GetServers: %w", result.Error)
	}
	return servers, nil
}

func (s *serverRepository) ImportServers(ctx context.Context, servers []model.Server) (insertedServers []model.Server, nonInsertedServers []model.Server, err error) {
	m := make(map[string]model.Server)
	for _, server := range servers {
		m[server.ServerName] = server
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		chunkSize := 1000
		for i := 0; i < len(servers); i += chunkSize {
			j := i + chunkSize
			if j > len(servers) {
				j = len(servers)
			}
			tempServers := servers[i:j]
			serversBatch := make([]model.Server, len(tempServers))
			copy(serversBatch, tempServers)
			result := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Clauses(clause.Returning{}).Create(&serversBatch)
			if result.Error != nil {
				return result.Error
			}
			for _, server := range serversBatch {
				insertedServers = append(insertedServers, server)
				delete(m, server.ServerName)
			}
		}
		return nil
	})
	if err != nil {
		err = fmt.Errorf("ServerRepository.ImportServers: %w", err)
		return
	}
	for _, server := range m {
		nonInsertedServers = append(nonInsertedServers, server)
	}
	return
}

func (s *serverRepository) GetServerById(ctx context.Context, serverId string) (model.Server, error) {
	var server model.Server
	result := s.db.WithContext(ctx).First(&server, "id = ?", serverId)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return server, fmt.Errorf("ServerRepository.GetServerById: %w", apperrors.ErrServerNotFound)
		}
		return server, fmt.Errorf("ServerRepository.GetServerById: %w", result.Error)
	}
	return server, nil
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
