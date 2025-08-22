package handler

import (
	"VCS_SMS_Microservice/internal/auth-service/api/dto/request"
	"VCS_SMS_Microservice/internal/auth-service/api/dto/response"
	"VCS_SMS_Microservice/internal/auth-service/api/middleware"
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"VCS_SMS_Microservice/internal/auth-service/service"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"go.uber.org/zap"
)

type UserHandler interface {
	GetUserByID() gin.HandlerFunc
	GetMe() gin.HandlerFunc
	UpdateUserRole() gin.HandlerFunc
	UpdateUserPassword() gin.HandlerFunc
	UpdateUserInfo() gin.HandlerFunc
	GetUsers() gin.HandlerFunc
}

type userHandler struct {
	userService service.UserService
	logger      Logger
}

func (u *userHandler) GetMe() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.Value(middleware.JWTClaimsContextKey).(jwt.MapClaims)
		userID := claims["user_id"].(string)
		user, err := u.userService.GetUserById(c, userID)
		if err != nil {
			switch {
			case errors.Is(err, apperrors.ErrUserNotFound):
				c.JSON(http.StatusNotFound, response.Response{
					Message: "User not found",
				})
			default:
				err = fmt.Errorf("userHandler.GetMe: %w", err)
				u.logger.LoggingError(c, err, "failed to get user info by id", zap.ErrorLevel)
				c.JSON(http.StatusInternalServerError, response.Response{
					Message: "Internal Server Error",
				})
			}
			return
		}
		scopeMap := make(map[string]response.ScopeInfoResponse)
		for _, role := range user.Roles {
			for _, scope := range role.Scopes {
				scopeMap[scope.ID] = response.ScopeInfoResponse{
					ID:          scope.ID,
					Name:        scope.Name,
					Description: scope.Description,
				}
			}
		}
		scopes := make([]response.ScopeInfoResponse, 0, len(scopeMap))
		for _, scope := range scopeMap {
			scopes = append(scopes, scope)
		}
		rolesRes := make([]response.RoleInfoResponse, len(user.Roles))
		for i, role := range user.Roles {
			rolesRes[i] = response.RoleInfoResponse{
				ID:          role.ID,
				Name:        role.Name,
				Description: role.Description,
			}
		}
		userRes := response.UserInfoResponse{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Roles:     rolesRes,
			Scopes:    scopes,
		}
		c.JSON(http.StatusOK, userRes)
	}
}

func (u *userHandler) GetUserByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		user, err := u.userService.GetUserById(c, id)
		if err != nil {
			switch {
			case errors.Is(err, apperrors.ErrUserNotFound):
				c.JSON(http.StatusNotFound, response.Response{
					Message: "User not found",
				})
			default:
				err = fmt.Errorf("userHandler.GetUserByID: %w", err)
				u.logger.LoggingError(c, err, "failed to get user info by id", zap.ErrorLevel)
				c.JSON(http.StatusInternalServerError, response.Response{
					Message: "Internal Server Error",
				})
			}
			return
		}
		scopeMap := make(map[string]response.ScopeInfoResponse)
		for _, role := range user.Roles {
			for _, scope := range role.Scopes {
				scopeMap[scope.ID] = response.ScopeInfoResponse{
					ID:          scope.ID,
					Name:        scope.Name,
					Description: scope.Description,
				}
			}
		}
		scopes := make([]response.ScopeInfoResponse, 0, len(scopeMap))
		for _, scope := range scopeMap {
			scopes = append(scopes, scope)
		}
		rolesRes := make([]response.RoleInfoResponse, len(user.Roles))
		for i, role := range user.Roles {
			rolesRes[i] = response.RoleInfoResponse{
				ID:          role.ID,
				Name:        role.Name,
				Description: role.Description,
			}
		}
		userRes := response.UserInfoResponse{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Roles:     rolesRes,
			Scopes:    scopes,
		}
		c.JSON(http.StatusOK, userRes)
	}
}

func (u *userHandler) UpdateUserRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req request.UpdateUserRoleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			var validatorError validator.ValidationErrors
			if errors.As(err, &validatorError) {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: u.formatValidationError(validatorError[0]),
				})
			} else {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Invalid request body",
				})
			}
			return
		}
		roles := make([]model.Role, len(req.RoleIDs))
		for i, roleID := range req.RoleIDs {
			roles[i] = model.Role{
				ID: roleID,
			}
		}
		updatedData := model.User{
			ID:    id,
			Roles: roles,
		}
		err := u.userService.UpdateUserByID(c, updatedData)
		if err != nil {
			switch {
			case errors.Is(err, apperrors.ErrUserNotFound):
				c.JSON(http.StatusNotFound, response.Response{
					Message: "User not found",
				})
			case errors.Is(err, apperrors.ErrInvalidRoles):
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Invalid roles",
				})
			default:
				err = fmt.Errorf("userHandler.UpdateUserRole: %w", err)
				u.logger.LoggingError(c, err, "failed to update user role by id", zap.ErrorLevel)
			}
			return
		}
		c.JSON(http.StatusOK, response.Response{
			Message: "User role updated",
		})
	}
}

func (u *userHandler) UpdateUserPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req request.UpdatePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			var validatorError validator.ValidationErrors
			if errors.As(err, &validatorError) {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: u.formatValidationError(validatorError[0]),
				})
			} else {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Invalid request body",
				})
			}
			return
		}
		claims := c.Value(middleware.JWTClaimsContextKey).(jwt.MapClaims)
		userId := claims["user_id"].(string)
		err := u.userService.UpdateUserPassword(c, userId, req.CurrentPassword, req.NewPassword)
		if err != nil {
			switch {
			case errors.Is(err, apperrors.ErrUserNotFound):
				c.JSON(http.StatusNotFound, response.Response{
					Message: "User not found",
				})
			case errors.Is(err, apperrors.ErrInvalidPassword):
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Invalid password",
				})
			default:
				u.logger.LoggingError(c, err, "failed to update user password", zap.ErrorLevel)
				c.JSON(http.StatusInternalServerError, response.Response{
					Message: "Internal server error",
				})
			}
			return
		}
		c.JSON(http.StatusOK, response.Response{
			Message: "Password updated successfully",
		})
	}
}

func (u *userHandler) UpdateUserInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req request.UpdateUserInfoRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			var validatorError validator.ValidationErrors
			if errors.As(err, &validatorError) {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: u.formatValidationError(validatorError[0]),
				})
			} else {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Invalid request body",
				})
			}
			return
		}
		claims := c.Value(middleware.JWTClaimsContextKey).(jwt.MapClaims)
		userId := claims["user_id"].(string)
		updatedData := model.User{
			ID:        userId,
			Email:     req.Email,
			FirstName: req.FirstName,
			LastName:  req.LastName,
		}
		err := u.userService.UpdateUserByID(c, updatedData)
		if err != nil {
			switch {
			case errors.Is(err, apperrors.ErrUserNotFound):
				c.JSON(http.StatusNotFound, response.Response{
					Message: "User not found",
				})
			case errors.Is(err, apperrors.ErrUserMailAlreadyExists):
				c.JSON(http.StatusConflict, response.Response{
					Message: "User mail already exists",
				})
			default:
				err = fmt.Errorf("userHandler.UpdateUserInfo: %w", err)
				u.logger.LoggingError(c, err, "failed to update user info by id", zap.ErrorLevel)
				c.JSON(http.StatusInternalServerError, response.Response{
					Message: "Internal server error",
				})
			}
			return
		}
		c.JSON(http.StatusOK, response.Response{
			Message: "User info updated successfully",
		})
	}
}

func (u *userHandler) GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		email := c.Query("email")
		offset := c.DefaultQuery("offset", "0")
		o, err := strconv.Atoi(offset)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Offset must be an integer",
			})
			return
		}
		limit := c.DefaultQuery("limit", "10")
		l, err := strconv.Atoi(limit)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Limit must be an integer",
			})
			return
		}
		if o < 0 {
			o = 0
		}
		if l <= 0 {
			l = 10
		}
		sortOrder := c.DefaultQuery("sort_order", "asc")
		if sortOrder != "asc" && sortOrder != "desc" {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Invalid sort order",
			})
			return
		}
		users, err := u.userService.GetUsers(c, email, sortOrder, l, o)
		if err != nil {
			err = fmt.Errorf("userHandler.GetUsers: %w", err)
			u.logger.LoggingError(c, err, "failed to get users", zap.ErrorLevel)
			c.JSON(http.StatusInternalServerError, response.Response{
				Message: "Internal server error",
			})
			return
		}
		usersRes := make([]response.UserInfoResponse, len(users))
		for i, user := range users {
			usersRes[i] = response.UserInfoResponse{
				ID:        user.ID,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Email:     user.Email,
			}
		}
		c.JSON(http.StatusOK, usersRes)
	}
}

func (u *userHandler) formatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("The %s field is required", err.Field())
	case "email":
		return fmt.Sprintf("The %s field is not a valid email", err.Field())
	default:
		return fmt.Sprintf("Validation failed for %s with tag %s.", err.Field(), err.Tag())
	}
}

func NewUserHandler(userService service.UserService, logger Logger) UserHandler {
	return &userHandler{
		userService: userService,
		logger:      logger,
	}
}
