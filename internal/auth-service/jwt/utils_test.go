package jwt

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	testSecretKey       = "this-is-a-super-secret-key-for-testing"
	testAccessTokenTTL  = 15 * time.Minute
	testRefreshTokenTTL = 7 * 24 * time.Hour
	testUserID          = "user-12345"
)

func TestUtils_CreateAccessToken(t *testing.T) {
	jwtUtils := NewJwtUtils(testSecretKey, testAccessTokenTTL, testRefreshTokenTTL)
	scopes := []string{"read:data", "write:data"}

	accessToken, err := jwtUtils.CreateAccessToken(testUserID, scopes...)

	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken.Token)
	assert.Equal(t, testAccessTokenTTL, accessToken.TTL)

	claims, err := jwtUtils.VerifyToken(accessToken.Token)
	assert.NoError(t, err)

	userIDClaim, ok := claims["user_id"].(string)
	assert.True(t, ok)
	assert.Equal(t, testUserID, userIDClaim)

	scopesClaim, ok := claims["scopes"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, scopesClaim, 2)

	var scopesFromToken []string
	for _, v := range scopesClaim {
		scopesFromToken = append(scopesFromToken, v.(string))
	}
	assert.ElementsMatch(t, scopes, scopesFromToken)

	expClaim, ok := claims["exp"].(float64)
	assert.True(t, ok)
	expireTime := time.Unix(int64(expClaim), 0)
	assert.True(t, expireTime.After(time.Now()))
	assert.WithinDuration(t, time.Now().Add(testAccessTokenTTL), expireTime, 5*time.Second)
}

func TestUtils_CreateRefreshToken(t *testing.T) {
	jwtUtils := NewJwtUtils(testSecretKey, testAccessTokenTTL, testRefreshTokenTTL)

	refreshToken, err := jwtUtils.CreateRefreshToken(testUserID)

	assert.NoError(t, err)
	assert.NotEmpty(t, refreshToken.Token)
	assert.NotEmpty(t, refreshToken.JTI)
	assert.Equal(t, testRefreshTokenTTL, refreshToken.TTL)

	claims, err := jwtUtils.VerifyToken(refreshToken.Token)
	assert.NoError(t, err)

	assert.Equal(t, testUserID, claims["user_id"])
	assert.Equal(t, refreshToken.JTI, claims["jti"])
}

func TestUtils_VerifyToken(t *testing.T) {
	jwtUtils := NewJwtUtils(testSecretKey, testAccessTokenTTL, testRefreshTokenTTL)

	t.Run("Success, Valid token", func(t *testing.T) {
		accessToken, _ := jwtUtils.CreateAccessToken(testUserID)
		claims, err := jwtUtils.VerifyToken(accessToken.Token)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, testUserID, claims["user_id"])
	})

	t.Run("Error, Invalid signature", func(t *testing.T) {
		jwtUtils1 := NewJwtUtils("secret-key-1", testAccessTokenTTL, testRefreshTokenTTL)
		accessToken, _ := jwtUtils1.CreateAccessToken(testUserID)

		jwtUtils2 := NewJwtUtils("secret-key-2", testAccessTokenTTL, testRefreshTokenTTL)
		_, err := jwtUtils2.VerifyToken(accessToken.Token)
		assert.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrInvalidToken)
	})

	t.Run("Error, Expired token", func(t *testing.T) {
		expiredClaims := jwt.MapClaims{
			"user_id": testUserID,
			"exp":     time.Now().Add(-1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
		expiredTokenString, _ := token.SignedString([]byte(testSecretKey))

		_, err := jwtUtils.VerifyToken(expiredTokenString)
		assert.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrInvalidToken)
	})

	t.Run("Error - Malformed token string", func(t *testing.T) {
		_, err := jwtUtils.VerifyToken("this.is.not.a.valid.token")
		assert.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrInvalidToken)
	})
}
