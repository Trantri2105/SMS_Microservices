package repository

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type RoleRepository interface {
	CreateRole(ctx context.Context, role model.Role) (model.Role, error)
	UpdateRoleByID(ctx context.Context, role model.Role) error
	DeleteRoleByID(ctx context.Context, id string) error
	GetRoles(ctx context.Context, roleName string, sortBy string, sortOrder string, limit int, offset int) ([]model.Role, error)
	GetRoleByID(ctx context.Context, id string) (model.Role, error)
	GetRolesListByIDs(ctx context.Context, ids []string) ([]model.Role, error)
}

type roleRepository struct {
	db *gorm.DB
}

func (r *roleRepository) GetRoleByID(ctx context.Context, id string) (model.Role, error) {
	var role model.Role
	result := r.db.WithContext(ctx).Preload("Scopes").First(&role, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return role, fmt.Errorf("roleRepository.GetRoleByID: %w", apperrors.ErrRoleNotFound)
		}
		return role, fmt.Errorf("roleRepository.GetRoleByID: %w", result.Error)
	}
	return role, nil
}

func (r *roleRepository) GetRolesListByIDs(ctx context.Context, ids []string) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&roles).Error
	if err != nil {
		return nil, fmt.Errorf("roleRepository.GetRolesListByIDs: %w", err)
	}
	return roles, nil
}

func (r *roleRepository) CreateRole(ctx context.Context, role model.Role) (model.Role, error) {
	err := r.db.WithContext(ctx).Omit("Scopes.*").Create(&role).Error
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			if pgErr.ConstraintName == "roles_name_key" {
				return role, fmt.Errorf("roleRepository.CreateRole: %w", apperrors.ErrRoleNameAlreadyExists)
			}
		}
		return role, fmt.Errorf("roleRepository.CreateRole: %w", err)
	}
	return role, nil
}

func (r *roleRepository) UpdateRoleByID(ctx context.Context, role model.Role) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Omit("ID", "Scopes").Updates(&role)
		if res.Error != nil {
			var pgErr *pgconn.PgError
			if errors.As(res.Error, &pgErr) && pgErr.Code == "23505" {
				if pgErr.ConstraintName == "roles_name_key" {
					return apperrors.ErrRoleNameAlreadyExists
				}
			}
			return res.Error
		}
		if res.RowsAffected == 0 {
			return apperrors.ErrRoleNotFound
		}
		if len(role.Scopes) > 0 {
			return tx.Model(&role).Omit("Scopes.*").Association("Scopes").Replace(&role.Scopes)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("roleRepository.UpdateRoleByID: %w", err)
	}
	return nil
}

func (r *roleRepository) DeleteRoleByID(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).Delete(&model.Role{}, id).Error
	if err != nil {
		return fmt.Errorf("roleRepository.DeleteRoleByID: %w", err)
	}
	return nil
}

func (r *roleRepository) GetRoles(ctx context.Context, roleName string, sortBy string, sortOrder string, limit int, offset int) ([]model.Role, error) {
	query := r.db.WithContext(ctx)
	if roleName != "" {
		query = query.Where("name LIKE ?", roleName+"%")
	}
	var roles []model.Role
	err := query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).Limit(limit).Offset(offset).Find(&roles).Error
	if err != nil {
		return nil, fmt.Errorf("roleRepository.GetRolesList: %w", err)
	}
	return roles, nil
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{
		db: db,
	}
}
