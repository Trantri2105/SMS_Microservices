package repository

import (
	"VCS_SMS_Microservice/internal/auth-service/model"
	"context"
	"fmt"

	"gorm.io/gorm"
)

type ScopeRepository interface {
	GetScopes(ctx context.Context, scopeName string, sortBy string, sortOrder string, limit, offset int) ([]model.Scope, error)
	GetScopesListByIDs(ctx context.Context, ids []string) ([]model.Scope, error)
}

type scopeRepository struct {
	db *gorm.DB
}

func (s *scopeRepository) GetScopesListByIDs(ctx context.Context, ids []string) ([]model.Scope, error) {
	var scopes []model.Scope
	err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&scopes).Error
	if err != nil {
		return nil, fmt.Errorf("scopeRepository.GetScopesListByID: %w", err)
	}
	return scopes, nil
}

func (s *scopeRepository) GetScopes(ctx context.Context, scopeName string, sortBy string, sortOrder string, limit, offset int) ([]model.Scope, error) {
	query := s.db.WithContext(ctx)
	if scopeName != "" {
		query = query.Where("name LIKE ?", scopeName+"%")
	}
	var scopes []model.Scope
	err := s.db.WithContext(ctx).Order(fmt.Sprintf("% %", sortBy, sortOrder)).Limit(limit).Offset(offset).Find(&scopes).Error
	if err != nil {
		return nil, fmt.Errorf("scopeRepository.GetScopes: %w", err)
	}
	return scopes, nil
}

func NewScopeRepository(db *gorm.DB) ScopeRepository {
	return &scopeRepository{
		db: db,
	}
}
