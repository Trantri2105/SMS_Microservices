package repository

import (
	"VCS_SMS_Microservice/internal/auth-service/model"
	"context"
	"fmt"

	"gorm.io/gorm"
)

type ScopeRepository interface {
	GetScopes(ctx context.Context, sortBy string, sortOrder string, limit, offset int) ([]model.Scope, error)
}

type scopeRepository struct {
	db *gorm.DB
}

func (s *scopeRepository) GetScopes(ctx context.Context, sortBy string, sortOrder string, limit, offset int) ([]model.Scope, error) {
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
