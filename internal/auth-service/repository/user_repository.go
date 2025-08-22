package repository

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user model.User) (model.User, error)
	GetUserByEmail(ctx context.Context, email string) (model.User, error)
	UpdateUserByID(ctx context.Context, user model.User) error
	// GetUsers sort users by CreatedAt
	GetUsers(ctx context.Context, userEmail string, sortOrder string, limit, offset int) ([]model.User, error)
	GetUserByID(ctx context.Context, id string) (model.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func (u *userRepository) CreateUser(ctx context.Context, user model.User) (model.User, error) {
	result := u.db.WithContext(ctx).Omit("Roles.*").Create(&user)
	if result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "users_email_key" {
				return user, fmt.Errorf("userRepository.CreateUser: %w", apperrors.ErrUserMailAlreadyExists)
			}
		}
		return model.User{}, fmt.Errorf("userRepository.CreateUser: %w", result.Error)
	}
	return user, nil
}

func (u *userRepository) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	var user model.User
	result := u.db.WithContext(ctx).Preload("Roles.Scopes").First(&user, "email = ?", email)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return user, fmt.Errorf("userRepository.GetUserByEmail: %w", apperrors.ErrUserNotFound)
		}
		return user, fmt.Errorf("userRepository.GetUserByEmail: %w", result.Error)
	}
	return user, nil
}

func (u *userRepository) UpdateUserByID(ctx context.Context, user model.User) error {
	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Omit("Roles").Updates(&user)
		if res.Error != nil {
			var pgErr *pgconn.PgError
			if errors.As(res.Error, &pgErr) && pgErr.Code == "23505" {
				if pgErr.ConstraintName == "users_email_key" {
					return apperrors.ErrUserMailAlreadyExists
				}
			}
			return res.Error
		}
		if res.RowsAffected == 0 {
			return apperrors.ErrUserNotFound
		}
		if len(user.Roles) > 0 {
			return tx.Model(&user).Omit("Roles.*").Association("Roles").Replace(user.Roles)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("userRepository.UpdateUserByID: %w", err)
	}
	return nil
}

func (u *userRepository) GetUsers(ctx context.Context, userEmail string, sortOrder string, limit, offset int) ([]model.User, error) {
	query := u.db.WithContext(ctx)
	if userEmail != "" {
		query = query.Where("email LIKE ?", userEmail+"%")
	}
	var users []model.User
	err := query.Order(fmt.Sprintf("%s %s", "created_at", sortOrder)).Limit(limit).Offset(offset).Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("userRepository.GetUsersList: %w", err)
	}
	return users, nil
}

func (u *userRepository) GetUserByID(ctx context.Context, id string) (model.User, error) {
	var user model.User
	result := u.db.WithContext(ctx).Preload("Roles.Scopes").First(&user, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return user, fmt.Errorf("userRepository.GetUserByID: %w", apperrors.ErrUserNotFound)
		}
		return user, fmt.Errorf("userRepository.GetUserByID: %w", result.Error)
	}
	return user, nil
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}
