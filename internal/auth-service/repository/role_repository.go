package repository

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoleRepository interface {
	CreateRole(ctx context.Context, role model.Role) (model.Role, error)
	UpdateRoleByID(ctx context.Context, role model.Role) (model.Role, error)
	DeleteRoleByID(ctx context.Context, id string) error
	GetRolesList(ctx context.Context, roleName string, sortBy string, sortOrder string, limit int, offset int) ([]model.Role, error)
}

type roleRepository struct {
	db *gorm.DB
}

func (r *roleRepository) CreateRole(ctx context.Context, role model.Role) (model.Role, error) {
	var createdRole model.Role
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(role.Scopes) != 0 {
			scopeName
			for _, scope := range role.Scopes {

			}
		}
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "roles_name_key" {
				return role, fmt.Errorf("roleRepository.CreateRole: %w", apperrors.ErrRoleNameAlreadyExists)
			}
		}
		return role, fmt.Errorf("roleRepository.CreateRole: %w", err)
	}
	return role, nil
}

func (r *roleRepository) UpdateRoleByID(ctx context.Context, role model.Role) (model.Role, error) {
	var updatedRole model.Role
	res := r.db.WithContext(ctx).Model(&updatedRole).Clauses(clause.Returning{}).Where("id = ?", role.ID).Updates(role)
	if res.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(res.Error, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "roles_name_key" {
				return updatedRole, fmt.Errorf("roleRepository.UpdateRoleByID: %w", apperrors.ErrRoleNameAlreadyExists)
			}
		}
		return updatedRole, fmt.Errorf("roleRepository.UpdateRoleByID: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return updatedRole, fmt.Errorf("roleRepository.UpdateRoleByID: %w", apperrors.ErrRoleNotFound)
	}
	return updatedRole, nil
}

func (r *roleRepository) DeleteRoleByID(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).Model(&model.Role{}).Where("id = ?", id).Delete(&model.Role{}).Error
	if err != nil {
		return fmt.Errorf("ServerRepository.DeleteRoleByID: %w", err)
	}
	return nil
}

func (r *roleRepository) GetRolesList(ctx context.Context, roleName string, sortBy string, sortOrder string, limit int, offset int) ([]model.Role, error) {
	query := r.db.WithContext(ctx)
	if roleName != "" {
		query = query.Where("name LIKE ?", roleName+"%")
	}
	var roles []model.Role
	err := query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).Limit(limit).Offset(offset).Find(&roles).Error
	if err != nil {
		return nil, fmt.Errorf("ServerRepository.GetRolesList: %w", err)
	}
	return roles, nil
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{
		db: db,
	}
}
