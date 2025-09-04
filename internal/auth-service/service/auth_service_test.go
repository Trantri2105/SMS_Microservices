package service

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	jwt2 "VCS_SMS_Microservice/internal/auth-service/jwt"
	mockjwt "VCS_SMS_Microservice/internal/auth-service/mocks/jwt"
	mockrepository "VCS_SMS_Microservice/internal/auth-service/mocks/repository"
	mockservice "VCS_SMS_Microservice/internal/auth-service/mocks/service"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
)

func TestAuthService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mockservice.NewMockUserService(ctrl)
	a := NewAuthService(mockUserService, nil, nil, 0)
	someErr := errors.New("some error")
	ctx := context.Background()
	userToRegister := model.User{Email: "test@example.com", Password: "password123"}
	registeredUser := model.User{ID: "1", Email: "test@example.com"}

	testCases := []struct {
		name      string
		input     model.User
		mock      func()
		output    model.User
		expectErr bool
	}{
		{
			name:  "Success",
			input: userToRegister,
			mock: func() {
				mockUserService.EXPECT().
					CreateUser(ctx, userToRegister).
					Return(registeredUser, nil).
					Times(1)
			},
			output:    registeredUser,
			expectErr: false,
		},
		{
			name:  "User service create user error",
			input: userToRegister,
			mock: func() {
				mockUserService.EXPECT().
					CreateUser(ctx, userToRegister).
					Return(model.User{}, someErr).
					Times(1)
			},
			output:    model.User{},
			expectErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()

			createdUser, err := a.Register(ctx, tc.input)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.output, createdUser)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mockservice.NewMockUserService(ctrl)
	mockJwtUtils := mockjwt.NewMockUtils(ctrl)
	mockTokenRepo := mockrepository.NewMockRefreshTokenRepository(ctrl)
	userSessionTTL := 15 * time.Minute
	a := NewAuthService(mockUserService, mockJwtUtils, mockTokenRepo, userSessionTTL)
	someErr := errors.New("some error")
	ctx := context.Background()
	email := "test@example.com"
	password := "password123"
	hashedPassword := hashPassword(password)
	userFromDB := model.User{
		ID:       "user1",
		Email:    email,
		Password: hashedPassword,
		Roles: []model.Role{
			{Name: "Admin", Scopes: []model.Scope{{Name: "users:read"}, {Name: "users:write"}}},
			{Name: "User", Scopes: []model.Scope{{Name: "users:read"}}},
		},
	}
	accessToken := jwt2.AccessToken{Token: "access_token", TTL: time.Minute * 15}
	refreshToken := jwt2.RefreshToken{Token: "refresh_token", JTI: "refresh_jti", TTL: time.Hour * 24}

	testCases := []struct {
		name      string
		email     string
		pass      string
		mock      func()
		output    AuthenticationResponse
		expectErr bool
	}{
		{
			name:  "Success",
			email: email,
			pass:  password,
			mock: func() {
				mockUserService.EXPECT().GetUserByEmail(ctx, email).Return(userFromDB, nil).Times(1)
				mockJwtUtils.EXPECT().CreateAccessToken("user1", gomock.InAnyOrder([]string{"users:read", "users:write"})).Return(accessToken, nil).Times(1)
				mockJwtUtils.EXPECT().CreateRefreshToken("user1").Return(refreshToken, nil).Times(1)
				mockTokenRepo.EXPECT().SetRefreshTokenID(ctx, "user1", "refresh_jti", userSessionTTL).Return(nil).Times(1)
			},
			output: AuthenticationResponse{
				AccessToken:     accessToken.Token,
				RefreshToken:    refreshToken.Token,
				AccessTokenTTL:  accessToken.TTL,
				RefreshTokenTTL: refreshToken.TTL,
			},
			expectErr: false,
		},
		{
			name:  "Get user by email error",
			email: email,
			pass:  password,
			mock: func() {
				mockUserService.EXPECT().GetUserByEmail(ctx, email).Return(model.User{}, someErr).Times(1)
			},
			expectErr: true,
		},
		{
			name:  "Invalid password",
			email: email,
			pass:  "wrong_password",
			mock: func() {
				mockUserService.EXPECT().GetUserByEmail(ctx, email).Return(userFromDB, nil).Times(1)
			},
			expectErr: true,
		},
		{
			name:  "Create access token error",
			email: email,
			pass:  password,
			mock: func() {
				mockUserService.EXPECT().GetUserByEmail(ctx, email).Return(userFromDB, nil).Times(1)
				mockJwtUtils.EXPECT().CreateAccessToken(gomock.Any(), gomock.Any()).Return(jwt2.AccessToken{}, someErr).Times(1)
			},
			expectErr: true,
		},
		{
			name:  "Create refresh token error",
			email: email,
			pass:  password,
			mock: func() {
				mockUserService.EXPECT().GetUserByEmail(ctx, email).Return(userFromDB, nil).Times(1)
				mockJwtUtils.EXPECT().CreateAccessToken(gomock.Any(), gomock.Any()).Return(accessToken, nil).Times(1)
				mockJwtUtils.EXPECT().CreateRefreshToken(gomock.Any()).Return(jwt2.RefreshToken{}, someErr).Times(1)
			},
			expectErr: true,
		},
		{
			name:  "Set refresh token error",
			email: email,
			pass:  password,
			mock: func() {
				mockUserService.EXPECT().GetUserByEmail(ctx, email).Return(userFromDB, nil).Times(1)
				mockJwtUtils.EXPECT().CreateAccessToken(gomock.Any(), gomock.Any()).Return(accessToken, nil).Times(1)
				mockJwtUtils.EXPECT().CreateRefreshToken(gomock.Any()).Return(refreshToken, nil).Times(1)
				mockTokenRepo.EXPECT().SetRefreshTokenID(ctx, "user1", "refresh_jti", userSessionTTL).Return(someErr).Times(1)
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			res, err := a.Login(ctx, tc.email, tc.pass)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.output, res)
			}
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTokenRepo := mockrepository.NewMockRefreshTokenRepository(ctrl)
	a := NewAuthService(nil, nil, mockTokenRepo, 0)
	someErr := errors.New("some error")
	ctx := context.Background()
	userID := "user1"

	testCases := []struct {
		name    string
		userID  string
		mock    func()
		wantErr bool
	}{
		{
			name:   "Success",
			userID: userID,
			mock: func() {
				mockTokenRepo.EXPECT().DeleteRefreshToken(ctx, userID).Return(nil).Times(1)
			},
			wantErr: false,
		},
		{
			name:   "Repository delete error",
			userID: userID,
			mock: func() {
				mockTokenRepo.EXPECT().DeleteRefreshToken(ctx, userID).Return(someErr).Times(1)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			err := a.Logout(ctx, tc.userID)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthService_Refresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mockservice.NewMockUserService(ctrl)
	mockJwtUtils := mockjwt.NewMockUtils(ctrl)
	mockTokenRepo := mockrepository.NewMockRefreshTokenRepository(ctrl)
	a := NewAuthService(mockUserService, mockJwtUtils, mockTokenRepo, 0)
	someErr := errors.New("some error")
	ctx := context.Background()
	refreshTokenString := "valid_refresh_token"
	userID := "user1"
	jti := "jti123"
	claims := map[string]interface{}{"user_id": userID, "jti": jti}

	userFromDB := model.User{ID: userID, Roles: []model.Role{{Scopes: []model.Scope{{Name: "users:read"}}}}}
	newAccessToken := jwt2.AccessToken{Token: "new_access_token", TTL: 15 * time.Minute}
	newRefreshToken := jwt2.RefreshToken{Token: "new_refresh_token", JTI: "new_jti", TTL: 24 * time.Hour}

	testCases := []struct {
		name    string
		token   string
		mock    func()
		want    AuthenticationResponse
		wantErr bool
	}{
		{
			name:  "Success",
			token: refreshTokenString,
			mock: func() {
				mockJwtUtils.EXPECT().VerifyToken(refreshTokenString).Return(claims, nil).Times(1)
				mockTokenRepo.EXPECT().GetRefreshTokenID(ctx, userID).Return(jti, nil).Times(1)
				mockUserService.EXPECT().GetUserById(ctx, userID).Return(userFromDB, nil).Times(1)
				mockJwtUtils.EXPECT().CreateAccessToken(userID, "users:read").Return(newAccessToken, nil).Times(1)
				mockJwtUtils.EXPECT().CreateRefreshToken(userID).Return(newRefreshToken, nil).Times(1)
				mockTokenRepo.EXPECT().SetRefreshTokenID(ctx, userID, newRefreshToken.JTI, time.Duration(-1)).Return(nil).Times(1)
			},
			want: AuthenticationResponse{
				AccessToken:     newAccessToken.Token,
				RefreshToken:    newRefreshToken.Token,
				AccessTokenTTL:  newAccessToken.TTL,
				RefreshTokenTTL: newRefreshToken.TTL,
			},
			wantErr: false,
		},
		{
			name:  "Verify token error",
			token: "invalid_token",
			mock: func() {
				mockJwtUtils.EXPECT().VerifyToken("invalid_token").Return(nil, apperrors.ErrInvalidToken).Times(1)
			},
			wantErr: true,
		},
		{
			name:  "GetRefreshTokenID error",
			token: refreshTokenString,
			mock: func() {
				mockJwtUtils.EXPECT().VerifyToken(refreshTokenString).Return(claims, nil).Times(1)
				mockTokenRepo.EXPECT().GetRefreshTokenID(ctx, userID).Return("", someErr).Times(1)
			},
			wantErr: true,
		},
		{
			name:  "JTI mismatch",
			token: refreshTokenString,
			mock: func() {
				mockJwtUtils.EXPECT().VerifyToken(refreshTokenString).Return(claims, nil).Times(1)
				mockTokenRepo.EXPECT().GetRefreshTokenID(ctx, userID).Return("different_jti", nil).Times(1)
				mockTokenRepo.EXPECT().DeleteRefreshToken(ctx, userID).Return(nil).Times(1)
			},
			wantErr: true,
		},
		{
			name:  "Delete token fails after JTI mismatch",
			token: refreshTokenString,
			mock: func() {
				mockJwtUtils.EXPECT().VerifyToken(refreshTokenString).Return(claims, nil).Times(1)
				mockTokenRepo.EXPECT().GetRefreshTokenID(ctx, userID).Return("different_jti", nil).Times(1)
				mockTokenRepo.EXPECT().DeleteRefreshToken(ctx, userID).Return(someErr).Times(1)
			},
			wantErr: true,
		},
		{
			name:  "GetUserById error",
			token: refreshTokenString,
			mock: func() {
				mockJwtUtils.EXPECT().VerifyToken(refreshTokenString).Return(claims, nil).Times(1)
				mockTokenRepo.EXPECT().GetRefreshTokenID(ctx, userID).Return(jti, nil).Times(1)
				mockUserService.EXPECT().GetUserById(ctx, userID).Return(model.User{}, someErr).Times(1)
			},
			wantErr: true,
		},
		{
			name:  "Create access token error",
			token: refreshTokenString,
			mock: func() {
				mockJwtUtils.EXPECT().VerifyToken(refreshTokenString).Return(claims, nil).Times(1)
				mockTokenRepo.EXPECT().GetRefreshTokenID(ctx, userID).Return(jti, nil).Times(1)
				mockUserService.EXPECT().GetUserById(ctx, userID).Return(userFromDB, nil).Times(1)
				mockJwtUtils.EXPECT().CreateAccessToken(userID, "users:read").Return(jwt2.AccessToken{}, someErr).Times(1)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			res, err := a.Refresh(ctx, tc.token)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, res)
			}
		})
	}
}
