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

type UserRepository interface {
	CreateUser(ctx context.Context, user model.User) (model.User, error)
	GetUserByEmail(ctx context.Context, email string) (model.User, error)
	UpdateUserByID(ctx context.Context, user model.User) (model.User, error)
	GetUsersList(ctx context.Context, userEmail string, sortBy string, sortOrder string, limit, offset int) ([]model.User, error)
	GetUserByID(ctx context.Context, id string) (model.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func (u *userRepository) CreateUser(ctx context.Context, user model.User) (model.User, error) {
	result := u.db.WithContext(ctx).Create(&user)
	if result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "users_email_key" {
				return user, fmt.Errorf("UserRepository.CreateUser: %w", apperrors.ErrUserMailAlreadyExists)
			}
		}
		return model.User{}, fmt.Errorf("UserRepository.CreateUser: %w", result.Error)
	}
	return user, nil
}

func (u *userRepository) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	var user model.User
	result := u.db.WithContext(ctx).First(&user, "email = ?", email)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return user, fmt.Errorf("UserRepository.GetUserByEmail: %w", apperrors.ErrUserNotFound)
		}
		return user, fmt.Errorf("UserRepository.GetUserByEmail: %w", result.Error)
	}
	return user, nil
}

func (u *userRepository) UpdateUserByID(ctx context.Context, user model.User) (model.User, error) {
	var updatedUser model.User
	result := u.db.WithContext(ctx).Model(&updatedUser).Clauses(clause.Returning{}).Where("id = ?", user.ID).Updates(user)
	if result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "users_email_key" {
				return updatedUser, fmt.Errorf("UserRepository.UpdateUserByID: %w", apperrors.ErrUserMailAlreadyExists)
			}
		}
		return updatedUser, fmt.Errorf("UserRepository.UpdateUserByID: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return user, fmt.Errorf("UserRepository.UpdateUserByID: %w", apperrors.ErrUserNotFound)
	}
	return user, nil
}

func (u *userRepository) GetUsersList(ctx context.Context, userEmail string, sortBy string, sortOrder string, limit, offset int) ([]model.User, error) {
	query := u.db.WithContext(ctx)
	if userEmail != "" {
		query = query.Where("email LIKE ?", userEmail+"%")
	}
	var users []model.User
	err := query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).Limit(limit).Offset(offset).Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("userRepository.GetUsersList: %w", err)
	}
	return users, nil
}

func (u *userRepository) GetUserByID(ctx context.Context, id string) (model.User, error) {
	var user model.User
	result := u.db.WithContext(ctx).First(&user, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return user, fmt.Errorf("UserRepository.GetUserByID: %w", apperrors.ErrUserNotFound)
		}
		return user, fmt.Errorf("UserRepository.GetUserByID: %w", result.Error)
	}
	return user, nil
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}
