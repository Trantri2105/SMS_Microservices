package apperrors

import (
	"errors"
)

var (
	ErrRoleNotFound          = errors.New("role not found")
	ErrRoleNameAlreadyExists = errors.New("role name already exists")
	ErrUserNotFound          = errors.New("user not found")
	ErrUserMailAlreadyExists = errors.New("user mail already exists")
	ErrInvalidToken          = errors.New("invalid token")
	ErrRefreshTokenNotFound  = errors.New("refresh token not found")
	ErrInvalidPassword       = errors.New("invalid password")
	ErrInvalidScopes         = errors.New("invalid scopes")
	ErrInvalidRoles          = errors.New("invalid roles")
)
