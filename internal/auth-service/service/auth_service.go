package service

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	"VCS_SMS_Microservice/internal/auth-service/jwt"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"VCS_SMS_Microservice/internal/auth-service/repository"
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthenticationResponse struct {
	AccessToken     string
	RefreshToken    string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type AuthUserInfo struct {
	UserID     string
	UserScopes []string
}

type AuthService interface {
	Register(ctx context.Context, user model.User) (model.User, error)
	Login(ctx context.Context, email, password string) (AuthenticationResponse, error)
	Logout(ctx context.Context, userID string) error
	Refresh(ctx context.Context, refreshToken string) (AuthenticationResponse, error)
	VerifyToken(ctx context.Context, token string) (AuthUserInfo, error)
}

type authService struct {
	userService    UserService
	jwt            jwt.Utils
	tokenRepo      repository.RefreshTokenRepository
	userSessionTTL time.Duration
}

func (a *authService) Register(ctx context.Context, user model.User) (model.User, error) {
	createdUser, err := a.userService.CreateUser(ctx, user)
	if err != nil {
		return model.User{}, fmt.Errorf("authService.Register: %w", err)
	}
	return createdUser, nil
}

func (a *authService) Login(ctx context.Context, email, password string) (AuthenticationResponse, error) {
	user, err := a.userService.GetUserByEmail(ctx, email)
	if err != nil {
		return AuthenticationResponse{}, fmt.Errorf("authService.Login: %w", err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return AuthenticationResponse{}, fmt.Errorf("authService.Login: %w", apperrors.ErrInvalidPassword)
	}
	scopeMap := make(map[string]struct{})
	for _, role := range user.Roles {
		for _, scope := range role.Scopes {
			scopeMap[scope.Name] = struct{}{}
		}
	}
	scopes := make([]string, 0, len(scopeMap))
	for scope := range scopeMap {
		scopes = append(scopes, scope)
	}
	accessToken, err := a.jwt.CreateAccessToken(user.ID, scopes...)
	if err != nil {
		return AuthenticationResponse{}, fmt.Errorf("authService.Login: %w", err)
	}
	refreshToken, err := a.jwt.CreateRefreshToken(user.ID)
	if err != nil {
		return AuthenticationResponse{}, fmt.Errorf("authService.Login: %w", err)
	}
	err = a.tokenRepo.SetRefreshTokenID(ctx, user.ID, refreshToken.JTI, a.userSessionTTL)
	if err != nil {
		return AuthenticationResponse{}, fmt.Errorf("authService.Login: %w", err)
	}
	res := AuthenticationResponse{
		AccessToken:     accessToken.Token,
		RefreshToken:    refreshToken.Token,
		AccessTokenTTL:  accessToken.TTL,
		RefreshTokenTTL: refreshToken.TTL,
	}
	return res, nil
}

func (a *authService) Logout(ctx context.Context, userID string) error {
	err := a.tokenRepo.DeleteRefreshToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("authService.Logout: %w", err)
	}
	return nil
}

func (a *authService) Refresh(ctx context.Context, refreshToken string) (AuthenticationResponse, error) {
	claims, err := a.jwt.VerifyToken(refreshToken)
	if err != nil {
		return AuthenticationResponse{}, fmt.Errorf("authService.Refresh: %w", err)
	}
	userID := claims["user_id"].(string)
	savedJTI, err := a.tokenRepo.GetRefreshTokenID(ctx, userID)
	if err != nil {
		return AuthenticationResponse{}, fmt.Errorf("authService.Refresh: %w", err)
	}
	if savedJTI != claims["jti"].(string) {
		err = a.tokenRepo.DeleteRefreshToken(ctx, userID)
		if err != nil {
			return AuthenticationResponse{}, fmt.Errorf("authService.Refresh: %w", err)
		}
		return AuthenticationResponse{}, fmt.Errorf("authService.Refresh: %w", apperrors.ErrInvalidToken)
	}
	user, err := a.userService.GetUserById(ctx, userID)
	if err != nil {
		return AuthenticationResponse{}, fmt.Errorf("authService.Refresh: %w", err)
	}
	scopeMap := make(map[string]struct{})
	for _, role := range user.Roles {
		for _, scope := range role.Scopes {
			scopeMap[scope.Name] = struct{}{}
		}
	}
	scopes := make([]string, 0, len(scopeMap))
	for scope := range scopeMap {
		scopes = append(scopes, scope)
	}
	accessToken, err := a.jwt.CreateAccessToken(user.ID, scopes...)
	if err != nil {
		return AuthenticationResponse{}, fmt.Errorf("AuthService.Refresh: %w", err)
	}
	newRefreshToken, err := a.jwt.CreateRefreshToken(user.ID)
	if err != nil {
		return AuthenticationResponse{}, fmt.Errorf("AuthService.Refresh: %w", err)
	}
	err = a.tokenRepo.SetRefreshTokenID(ctx, user.ID, newRefreshToken.JTI, -1)
	if err != nil {
		return AuthenticationResponse{}, fmt.Errorf("AuthService.Refresh: %w", err)
	}
	return AuthenticationResponse{
		AccessToken:     accessToken.Token,
		RefreshToken:    newRefreshToken.Token,
		AccessTokenTTL:  accessToken.TTL,
		RefreshTokenTTL: newRefreshToken.TTL,
	}, nil
}

func (a *authService) VerifyToken(ctx context.Context, token string) (AuthUserInfo, error) {
	//TODO implement me
	panic("implement me")
}

func NewAuthService(userService UserService, jwt jwt.Utils, tokenRepo repository.RefreshTokenRepository, userSessionTTL time.Duration) AuthService {
	return &authService{
		userService:    userService,
		jwt:            jwt,
		tokenRepo:      tokenRepo,
		userSessionTTL: userSessionTTL,
	}
}
