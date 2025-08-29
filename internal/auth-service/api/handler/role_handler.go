package handler

import (
	"VCS_SMS_Microservice/internal/auth-service/api/dto/request"
	"VCS_SMS_Microservice/internal/auth-service/api/dto/response"
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"VCS_SMS_Microservice/internal/auth-service/service"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type RoleHandler interface {
	CreateRole() gin.HandlerFunc
	UpdateRole() gin.HandlerFunc
	DeleteRole() gin.HandlerFunc
	GetRoles() gin.HandlerFunc
	GetRoleByID() gin.HandlerFunc
}

type roleHandler struct {
	roleService service.RoleService
	logger      Logger
}

func (*roleHandler) formatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("The %s field is required", err.Field())
	default:
		return fmt.Sprintf("Validation failed for %s with tag %s.", err.Field(), err.Tag())
	}
}

func (r *roleHandler) CreateRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req request.CreateRoleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			var validatorError validator.ValidationErrors
			if errors.As(err, &validatorError) {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: r.formatValidationError(validatorError[0]),
				})
			} else {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Invalid request body",
				})
			}
			return
		}
		scopes := make([]model.Scope, len(req.ScopeIDs))
		for i, scope := range req.ScopeIDs {
			scopes[i] = model.Scope{
				ID: scope,
			}
		}
		newRole := model.Role{
			Name:        req.Name,
			Description: req.Description,
			Scopes:      scopes,
		}
		res, err := r.roleService.CreateRole(c, newRole)
		if err != nil {
			switch {
			case errors.Is(err, apperrors.ErrInvalidScopes):
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Invalid scopes",
				})
			case errors.Is(err, apperrors.ErrRoleNameAlreadyExists):
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Role name already exists",
				})
			default:
				err = fmt.Errorf("roleHandler.CreateRole: %w", err)
				r.logger.LoggingError(c, err, "failed to create new role", zap.ErrorLevel)
				c.JSON(http.StatusInternalServerError, response.Response{
					Message: "Internal server error",
				})
			}
			return
		}
		scopeRes := make([]response.ScopeInfoResponse, len(res.Scopes))
		for i, scope := range res.Scopes {
			scopeRes[i] = response.ScopeInfoResponse{
				Name: scope.Name,
				ID:   scope.ID,
			}
		}
		c.JSON(http.StatusCreated, response.RoleInfoResponse{
			ID:          res.ID,
			Name:        res.Name,
			Description: res.Description,
			Scopes:      scopeRes,
		})
	}
}

func (r *roleHandler) UpdateRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req request.UpdateRoleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			var validatorError validator.ValidationErrors
			if errors.As(err, &validatorError) {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: r.formatValidationError(validatorError[0]),
				})
			} else {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Invalid request body",
				})
			}
			return
		}
		id := c.Param("id")
		scopes := make([]model.Scope, len(req.ScopeIDs))
		for i, scope := range req.ScopeIDs {
			scopes[i] = model.Scope{
				Name: scope,
			}
		}
		updatedData := model.Role{
			ID:          id,
			Name:        req.Name,
			Description: req.Description,
			Scopes:      scopes,
		}
		err := r.roleService.UpdateRoleByID(c, updatedData)
		if err != nil {
			switch {
			case errors.Is(err, apperrors.ErrInvalidScopes):
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Invalid scopes",
				})
			case errors.Is(err, apperrors.ErrRoleNotFound):
				c.JSON(http.StatusNotFound, response.Response{
					Message: "Role not found",
				})
			case errors.Is(err, apperrors.ErrRoleNameAlreadyExists):
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Role name already exists",
				})
			default:
				err = fmt.Errorf("roleHandler.UpdateRole: %w", err)
				r.logger.LoggingError(c, err, "failed to update role", zap.ErrorLevel)
				c.JSON(http.StatusInternalServerError, response.Response{
					Message: "Internal server error",
				})
			}
			return
		}
		c.JSON(http.StatusOK, response.Response{
			Message: "Role updated",
		})
	}
}

func (r *roleHandler) DeleteRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		err := r.roleService.DeleteRoleByID(c, id)
		if err != nil {
			err = fmt.Errorf("roleHandler.DeleteRole: %w", err)
			r.logger.LoggingError(c, err, "failed to delete role", zap.ErrorLevel)
			c.JSON(http.StatusInternalServerError, response.Response{
				Message: "Internal server error",
			})
			return
		}
		c.JSON(http.StatusOK, response.Response{
			Message: "Role deleted",
		})
	}
}

func (r *roleHandler) GetRoles() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleName := c.Query("role_name")
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
		sortBy := c.DefaultQuery("sort_by", "created_at")
		if sortBy != "name" && sortBy != "created_at" {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Invalid sort by",
			})
			return
		}
		sortOrder := c.DefaultQuery("sort_order", "asc")
		if sortOrder != "asc" && sortOrder != "desc" {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Invalid sort order",
			})
			return
		}
		roles, err := r.roleService.GetRoles(c, roleName, sortBy, sortOrder, l, o)
		if err != nil {
			err = fmt.Errorf("roleHandler.GetRoles: %w", err)
			r.logger.LoggingError(c, err, "failed to fetch roles", zap.ErrorLevel)
			c.JSON(http.StatusInternalServerError, response.Response{
				Message: "Internal server error",
			})
			return
		}
		rolesRes := make([]response.RoleInfoResponse, len(roles))
		for i, role := range roles {
			rolesRes[i] = response.RoleInfoResponse{
				ID:          role.ID,
				Name:        role.Name,
				Description: role.Description,
			}
		}
		c.JSON(http.StatusOK, rolesRes)
	}
}

func (r *roleHandler) GetRoleByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		role, err := r.roleService.GetRoleByID(c, id)
		if err != nil {
			switch {
			case errors.Is(err, apperrors.ErrRoleNotFound):
				c.JSON(http.StatusNotFound, response.Response{
					Message: "Role not found",
				})
			default:
				err = fmt.Errorf("roleHandler.GetRoleByID: %w", err)
				r.logger.LoggingError(c, err, "failed to get role", zap.ErrorLevel)
				c.JSON(http.StatusInternalServerError, response.Response{
					Message: "Internal server error",
				})
			}
			return
		}
		scopesRes := make([]response.ScopeInfoResponse, len(role.Scopes))
		for i, scope := range role.Scopes {
			scopesRes[i] = response.ScopeInfoResponse{
				Name:        scope.Name,
				ID:          scope.ID,
				Description: scope.Description,
			}
		}
		c.JSON(http.StatusOK, response.RoleInfoResponse{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
			Scopes:      scopesRes,
		})
	}
}

func NewRoleHandler(roleService service.RoleService, logger Logger) RoleHandler {
	return &roleHandler{
		roleService: roleService,
		logger:      logger,
	}
}
