package service

import (
	"VCS_SMS_Microservice/internal/auth-service/model"
	"VCS_SMS_Microservice/internal/auth-service/repository"
	"context"
	"fmt"
)

type ScopeService interface {
	GetScopesList(ctx context.Context, scopeName string, sortBy string, sortOrder string, limit, offset int) ([]model.Scope, error)
	GetScopesByIDs(ctx context.Context, ids []string) ([]model.Scope, error)
}

type scopeService struct {
	repository repository.ScopeRepository
}

func (s *scopeService) GetScopesByIDs(ctx context.Context, ids []string) ([]model.Scope, error) {
	scopes, err := s.repository.GetScopesListByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("scopeService.GetScopesByIDs: %w", err)
	}
	return scopes, nil
}

func (s *scopeService) GetScopesList(ctx context.Context, scopeName string, sortBy string, sortOrder string, limit, offset int) ([]model.Scope, error) {
	scopes, err := s.repository.GetScopes(ctx, scopeName, sortBy, sortOrder, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("scopeService.GetScopesList: %w", err)
	}
	return scopes, nil
}

func NewScopeService(repository repository.ScopeRepository) ScopeService {
	return &scopeService{
		repository: repository,
	}
}
