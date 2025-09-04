package jwt

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

type AccessToken struct {
	Token string
	TTL   time.Duration
}

type RefreshToken struct {
	Token string
	TTL   time.Duration
	JTI   string
}

type Utils interface {
	CreateAccessToken(userID string, scopes ...string) (AccessToken, error)
	CreateRefreshToken(userID string) (RefreshToken, error)
	VerifyToken(tokenString string) (jwt.MapClaims, error)
}

type utils struct {
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	secretKey       string
}

func (u *utils) CreateAccessToken(userId string, scopes ...string) (AccessToken, error) {
	expireTime := time.Now().Add(u.accessTokenTTL).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userId,
		"scopes":  scopes,
		"exp":     expireTime,
	})
	tokenString, err := token.SignedString([]byte(u.secretKey))
	if err != nil {
		return AccessToken{}, fmt.Errorf("jwt.utils.CreateToken signing token: %w", err)
	}
	return AccessToken{
		Token: tokenString,
		TTL:   u.accessTokenTTL,
	}, nil
}

func (u *utils) CreateRefreshToken(userId string) (RefreshToken, error) {
	expireTime := time.Now().Add(u.refreshTokenTTL).Unix()
	jti, err := uuid.NewRandom()
	if err != nil {
		return RefreshToken{}, fmt.Errorf("jwt.utils.CreateRefreshToken: %w", err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"jti":     jti.String(),
		"user_id": userId,
		"exp":     expireTime,
	})
	tokenString, err := token.SignedString([]byte(u.secretKey))
	if err != nil {
		return RefreshToken{}, fmt.Errorf("jwt.utils.CreateToken signing token: %w", err)
	}
	return RefreshToken{
		Token: tokenString,
		TTL:   u.refreshTokenTTL,
		JTI:   jti.String(),
	}, nil
}

func (u *utils) VerifyToken(tokenString string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	parsedToken, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt.Utils.VerifyToken: %w", apperrors.ErrInvalidToken)
		}
		return []byte(u.secretKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwt.Utils.VerifyToken: %w", apperrors.ErrInvalidToken)
	}
	if !parsedToken.Valid {
		return nil, fmt.Errorf("jwt.Utils.VerifyToken: %w", apperrors.ErrInvalidToken)
	}
	return claims, nil
}

func NewJwtUtils(secretKey string, accessTokenTTL, refreshTokenTTL time.Duration) Utils {
	return &utils{
		secretKey:       secretKey,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}
