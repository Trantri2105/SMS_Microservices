package handler

import (
	"VCS_SMS_Microservice/internal/auth-service/api/dto/response"
	"VCS_SMS_Microservice/internal/auth-service/service"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ScopeHandler interface {
	GetScopes() gin.HandlerFunc
}

type scopeHandler struct {
	service service.ScopeService
	logger  Logger
}

func (s *scopeHandler) GetScopes() gin.HandlerFunc {
	return func(c *gin.Context) {
		scopeName := c.Query("scope_name")
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
		scopes, err := s.service.GetScopesList(c, scopeName, sortBy, sortOrder, l, o)
		if err != nil {
			err = fmt.Errorf("scopeHandler.GetScopes: %w", err)
			s.logger.LoggingError(c, err, "failed to get scopes", zap.ErrorLevel)
			c.JSON(http.StatusInternalServerError, response.Response{
				Message: "Internal Server Error",
			})
			return
		}
		scopesRes := make([]response.ScopeResponse, len(scopes))
		for i, scope := range scopes {
			scopesRes[i] = response.ScopeResponse{
				ID:          scope.ID,
				Name:        scope.Name,
				Description: scope.Description,
			}
		}
		c.JSON(http.StatusOK, scopesRes)
	}
}

func NewScopeHandler(service service.ScopeService, logger Logger) ScopeHandler {
	return &scopeHandler{
		service: service,
		logger:  logger,
	}
}
