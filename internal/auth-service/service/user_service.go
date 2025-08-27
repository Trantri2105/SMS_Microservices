package service

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"VCS_SMS_Microservice/internal/auth-service/repository"
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	CreateUser(ctx context.Context, user model.User) (model.User, error)
	GetUserByEmail(ctx context.Context, email string) (model.User, error)
	GetUserById(ctx context.Context, id string) (model.User, error)
	UpdateUserByID(ctx context.Context, user model.User) error
	UpdateUserPassword(ctx context.Context, id string, currentPassword string, newPassword string) error
	// GetUsers will sort user by CreatedAt
	GetUsers(ctx context.Context, userEmail string, sortOrder string, limit, offset int) ([]model.User, error)
}

type userService struct {
	userRepo    repository.UserRepository
	roleService RoleService
}

func (u *userService) GetUsers(ctx context.Context, userEmail string, sortOrder string, limit, offset int) ([]model.User, error) {
	users, err := u.userRepo.GetUsers(ctx, userEmail, sortOrder, limit, offset)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (u *userService) CreateUser(ctx context.Context, user model.User) (model.User, error) {
	if len(user.Roles) > 0 {
		roleMap := make(map[string]struct{})
		for _, role := range user.Roles {
			roleMap[role.ID] = struct{}{}
		}
		roleIDs := make([]string, 0, len(roleMap))
		for roleID := range roleMap {
			roleIDs = append(roleIDs, roleID)
		}
		roles, err := u.roleService.GetRolesByIDs(ctx, roleIDs)
		if err != nil {
			return model.User{}, fmt.Errorf("userRepository.GetRolesListByIDs: %w", err)
		}
		if len(roles) != len(roleIDs) {
			return model.User{}, fmt.Errorf("userRepository.GetRolesListByIDs: %w", apperrors.ErrInvalidRoles)
		}
		user.Roles = roles
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, fmt.Errorf("UserService.Register hashing password err: %w", err)
	}
	user.Password = string(hash)
	createdUser, err := u.userRepo.CreateUser(ctx, user)
	if err != nil {
		return model.User{}, fmt.Errorf("userService.CreateUser: %w", err)
	}
	return createdUser, nil
}

func (u *userService) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	user, err := u.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return model.User{}, fmt.Errorf("userService.GetUserByEmail: %w", err)
	}
	return user, nil
}

func (u *userService) GetUserById(ctx context.Context, id string) (model.User, error) {
	user, err := u.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return model.User{}, fmt.Errorf("userService.GetUserById: %w", err)
	}
	return user, nil
}

func (u *userService) UpdateUserByID(ctx context.Context, user model.User) error {
	if len(user.Roles) > 0 {
		roleMap := make(map[string]bool)
		for _, role := range user.Roles {
			roleMap[role.ID] = true
		}
		roleIDs := make([]string, 0, len(roleMap))
		for roleID := range roleMap {
			roleIDs = append(roleIDs, roleID)
		}
		roles, err := u.roleService.GetRolesByIDs(ctx, roleIDs)
		if err != nil {
			return fmt.Errorf("userRepository.GetRolesListByIDs: %w", err)
		}
		if len(roles) != len(roleIDs) {
			return fmt.Errorf("userRepository.GetRolesListByIDs: %w", apperrors.ErrInvalidRoles)
		}
		user.Roles = roles
	}
	err := u.userRepo.UpdateUserByID(ctx, user)
	if err != nil {
		return fmt.Errorf("userService.UpdateUserByID: %w", err)
	}
	return nil
}

func (u *userService) UpdateUserPassword(ctx context.Context, id string, currentPassword string, newPassword string) error {
	user, err := u.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return fmt.Errorf("userService.UpdateUserPassword: %w", err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword))
	if err != nil {
		return fmt.Errorf("userService.UpdateUserPassword: %w", apperrors.ErrInvalidPassword)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("userService.UpdateUserPassword hashing password: %w", err)
	}
	user.Password = string(hash)
	err = u.userRepo.UpdateUserByID(ctx, model.User{ID: id, Password: user.Password})
	if err != nil {
		return fmt.Errorf("userService.UpdateUserPassword: %w", err)
	}
	return nil
}

func NewUserService(userRepo repository.UserRepository, roleService RoleService) UserService {
	return &userService{
		userRepo:    userRepo,
		roleService: roleService,
	}
}
