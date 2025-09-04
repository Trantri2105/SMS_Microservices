package repository

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func newTestRefreshTokenRepoWithMockRedis() (RefreshTokenRepository, redismock.ClientMock) {
	db, mock := redismock.NewClientMock()
	repo := NewRefreshTokenRepository(db)
	return repo, mock
}

func TestRefreshTokenRepository_SetRefreshTokenID(t *testing.T) {
	userID := "user-123"
	tokenID := "token-abc"
	key := fmt.Sprintf("user:%s", userID)

	tests := []struct {
		name        string
		ttl         time.Duration
		mockSetup   func(mock redismock.ClientMock)
		expectError bool
	}{
		{
			name: "Success, Set with TTL",
			ttl:  10 * time.Minute,
			mockSetup: func(mock redismock.ClientMock) {
				mock.ExpectSet(key, tokenID, 10*time.Minute).SetVal("OK")
			},
			expectError: false,
		},
		{
			name: "Success, Set without expiration",
			ttl:  0,
			mockSetup: func(mock redismock.ClientMock) {
				mock.ExpectSet(key, tokenID, 0).SetVal("OK")
			},
			expectError: false,
		},
		{
			name: "Error - Redis returns an error",
			ttl:  10 * time.Minute,
			mockSetup: func(mock redismock.ClientMock) {
				mock.ExpectSet(key, tokenID, 10*time.Minute).SetErr(errors.New("redis connection error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestRefreshTokenRepoWithMockRedis()
			tt.mockSetup(mock)

			err := repo.SetRefreshTokenID(context.Background(), userID, tokenID, tt.ttl)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRefreshTokenRepository_GetRefreshTokenID(t *testing.T) {
	userID := "user-456"
	tokenID := "token-xyz"
	key := fmt.Sprintf("user:%s", userID)
	redisError := errors.New("redis error")
	tests := []struct {
		name          string
		mockSetup     func(mock redismock.ClientMock)
		expectedToken string
		expectedError error
	}{
		{
			name: "Success, Token found",
			mockSetup: func(mock redismock.ClientMock) {
				mock.ExpectGet(key).SetVal(tokenID)
			},
			expectedToken: tokenID,
		},
		{
			name: "Error, Token not found (redis.Nil)",
			mockSetup: func(mock redismock.ClientMock) {
				mock.ExpectGet(key).RedisNil()
			},
			expectedToken: "",
			expectedError: apperrors.ErrRefreshTokenNotFound,
		},
		{
			name: "Error, Generic Redis error",
			mockSetup: func(mock redismock.ClientMock) {
				mock.ExpectGet(key).SetErr(redisError)
			},
			expectedToken: "",
			expectedError: redisError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestRefreshTokenRepoWithMockRedis()
			tt.mockSetup(mock)

			actualToken, err := repo.GetRefreshTokenID(context.Background(), userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, actualToken)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRefreshTokenRepository_DeleteRefreshToken(t *testing.T) {
	userID := "user-789"
	key := fmt.Sprintf("user:%s", userID)

	tests := []struct {
		name        string
		mockSetup   func(mock redismock.ClientMock)
		expectError bool
	}{
		{
			name: "Success - Key deleted",
			mockSetup: func(mock redismock.ClientMock) {
				mock.ExpectDel(key).SetVal(1)
			},
			expectError: false,
		},
		{
			name: "Success - Key did not exist but no error",
			mockSetup: func(mock redismock.ClientMock) {
				mock.ExpectDel(key).SetVal(0)
			},
			expectError: false,
		},
		{
			name: "Error - Redis returns an error",
			mockSetup: func(mock redismock.ClientMock) {
				mock.ExpectDel(key).SetErr(errors.New("redis command failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestRefreshTokenRepoWithMockRedis()
			tt.mockSetup(mock)

			err := repo.DeleteRefreshToken(context.Background(), userID)

			assert.Equal(t, tt.expectError, err != nil)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetUserRefreshTokenKey(t *testing.T) {
	repo := &refreshTokenRepository{}
	userID := "test-id"
	expectedKey := "user:test-id"
	actualKey := repo.getUserRefreshTokenKey(userID)
	assert.Equal(t, expectedKey, actualKey)
}
