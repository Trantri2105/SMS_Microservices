package repository

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RefreshTokenRepository interface {
	// SetRefreshTokenID set refresh token id with expiration time
	//
	// Set ttl parameter to -1 to keep existing TTL and 0 to have no expiration time
	SetRefreshTokenID(ctx context.Context, userID string, refreshTokenID string, ttl time.Duration) error
	GetRefreshTokenID(ctx context.Context, userID string) (string, error)
	DeleteRefreshToken(ctx context.Context, userID string) error
}

type refreshTokenRepository struct {
	redis *redis.Client
}

func (*refreshTokenRepository) getUserRefreshTokenKey(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

func (r *refreshTokenRepository) SetRefreshTokenID(ctx context.Context, userID string, refreshTokenID string, ttl time.Duration) error {
	key := r.getUserRefreshTokenKey(userID)
	err := r.redis.Set(ctx, key, refreshTokenID, ttl).Err()
	if err != nil {
		return fmt.Errorf("refreshTokenRepository.SetRefreshTokenID: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) GetRefreshTokenID(ctx context.Context, userID string) (string, error) {
	key := r.getUserRefreshTokenKey(userID)
	tokenID, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", fmt.Errorf("refreshTokenRepository.GetRefreshTokenID: %w", apperrors.ErrRefreshTokenNotFound)
		}
		return "", fmt.Errorf("refreshTokenRepository.GetRefreshTokenID: %w", err)
	}
	return tokenID, nil
}

func (r *refreshTokenRepository) DeleteRefreshToken(ctx context.Context, userID string) error {
	key := r.getUserRefreshTokenKey(userID)
	err := r.redis.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("refreshTokenRepository.DeleteRefreshToken: %w", err)
	}
	return nil
}

func NewRefreshTokenRepository(redis *redis.Client) RefreshTokenRepository {
	return &refreshTokenRepository{
		redis: redis,
	}
}
