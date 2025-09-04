package service

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"VCS_SMS_Microservice/internal/auth-service/repository"
	"context"
	"fmt"
)

type RoleService interface {
	CreateRole(ctx context.Context, role model.Role) (model.Role, error)
	UpdateRoleByID(ctx context.Context, role model.Role) error
	DeleteRoleByID(ctx context.Context, id string) error
	GetRoles(ctx context.Context, roleName string, sortBy string, sortOrder string, limit int, offset int) ([]model.Role, error)
	GetRoleByID(ctx context.Context, id string) (model.Role, error)
	GetRoleListByIDs(ctx context.Context, ids []string) ([]model.Role, error)
}

type roleService struct {
	roleRepo     repository.RoleRepository
	scopeService ScopeService
}

func (r *roleService) GetRoleListByIDs(ctx context.Context, ids []string) ([]model.Role, error) {
	roles, err := r.roleRepo.GetRolesListByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("roleSerivce.GetRoleListByIDs: %w", err)
	}
	return roles, nil
}

func (r *roleService) validateScopes(ctx context.Context, scopes []model.Scope) ([]model.Scope, error) {
	scopeMap := make(map[string]struct{})
	for _, scope := range scopes {
		scopeMap[scope.ID] = struct{}{}
	}
	scopeIDs := make([]string, 0, len(scopeMap))
	for scopeID := range scopeMap {
		scopeIDs = append(scopeIDs, scopeID)
	}
	scopeInfos, err := r.scopeService.GetScopesByIDs(ctx, scopeIDs)
	if err != nil {
		return nil, fmt.Errorf("roleService.GetScopesListByIDs: %w", err)
	}
	if len(scopeInfos) != len(scopeIDs) {
		return nil, fmt.Errorf("roleService.GetScopesListByIDs: %w", apperrors.ErrInvalidScopes)
	}
	return scopeInfos, nil
}

func (r *roleService) CreateRole(ctx context.Context, role model.Role) (model.Role, error) {
	if len(role.Scopes) > 0 {
		scopes, err := r.validateScopes(ctx, role.Scopes)
		if err != nil {
			return model.Role{}, fmt.Errorf("roleService.CreateRole: %w", err)
		}
		role.Scopes = scopes
	}
	createdRole, err := r.roleRepo.CreateRole(ctx, role)
	if err != nil {
		return model.Role{}, fmt.Errorf("roleService.CreateRole: %w", err)
	}
	return createdRole, nil
}

func (r *roleService) UpdateRoleByID(ctx context.Context, role model.Role) error {
	if len(role.Scopes) > 0 {
		scopes, err := r.validateScopes(ctx, role.Scopes)
		if err != nil {
			return fmt.Errorf("roleService.UpdateRoleByID: %w", err)
		}
		role.Scopes = scopes
	}
	err := r.roleRepo.UpdateRoleByID(ctx, role)
	if err != nil {
		return fmt.Errorf("roleService.UpdateRoleByID: %w", err)
	}
	return nil
}

func (r *roleService) DeleteRoleByID(ctx context.Context, id string) error {
	err := r.roleRepo.DeleteRoleByID(ctx, id)
	if err != nil {
		return fmt.Errorf("roleService.DeleteRoleByID: %w", err)
	}
	return nil
}

func (r *roleService) GetRoles(ctx context.Context, roleName string, sortBy string, sortOrder string, limit int, offset int) ([]model.Role, error) {
	roles, err := r.roleRepo.GetRoles(ctx, roleName, sortBy, sortOrder, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("roleService.GetRoles: %w", err)
	}
	return roles, nil
}

func (r *roleService) GetRoleByID(ctx context.Context, id string) (model.Role, error) {
	role, err := r.roleRepo.GetRoleByID(ctx, id)
	if err != nil {
		return model.Role{}, fmt.Errorf("roleService.GetRoleByID: %w", err)
	}
	return role, nil
}

func NewRoleService(roleRepo repository.RoleRepository, scopeService ScopeService) RoleService {
	return &roleService{
		roleRepo:     roleRepo,
		scopeService: scopeService,
	}
}
