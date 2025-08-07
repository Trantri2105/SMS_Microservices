package handler

import (
	"VCS_SMS_Microservice/internal/server-service/api/dto/request"
	"VCS_SMS_Microservice/internal/server-service/api/dto/response"
	apperrors "VCS_SMS_Microservice/internal/server-service/errors"
	"VCS_SMS_Microservice/internal/server-service/model"
	"VCS_SMS_Microservice/internal/server-service/service"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ServerHandler interface {
	CreateServer() gin.HandlerFunc
	ImportServersFromExcelFile() gin.HandlerFunc
	ExportServersToExcelFile() gin.HandlerFunc
	ReportAllServersHealthInfo() gin.HandlerFunc
	UpdateServer() gin.HandlerFunc
	DeleteServer() gin.HandlerFunc
	GetServers() gin.HandlerFunc
	GetServerUptimePercentage() gin.HandlerFunc
}

type serverHandler struct {
	logger        *zap.Logger
	serverService service.ServerService
	validator     *validator.Validate
}

func (*serverHandler) formatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("The %s field is required", err.Field())
	case "email":
		return fmt.Sprintf("The %s field is not a valid email", err.Field())
	case "datetime":
		return fmt.Sprintf("The %s field is not a valid datetime, use YYYY-MM-DD format", err.Field())
	case "gte":
		return fmt.Sprintf("The %s field must be greater than or equal to %s", err.Field(), err.Param())
	case "ipv4":
		return fmt.Sprintf("The %s field is not a valid ipv4", err.Field())
	default:
		return fmt.Sprintf("Validation failed for %s with tag %s.", err.Field(), err.Tag())
	}
}

func (s *serverHandler) GetServerUptimePercentage() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		startDate := c.Query("start_date")
		startTime, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Invalid start date",
			})
			return
		}
		endDate := c.Query("end_date")
		endTime, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Invalid end date",
			})
			return
		}
		if endTime.Before(startTime) {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Invalid end date",
			})
			return
		}
		endTimeFinal := endTime.AddDate(0, 0, 1)
		res, err := s.serverService.GetServerUptimePercentage(c, id, startTime, endTimeFinal)
		if err != nil {
			err = fmt.Errorf("ServerHandler.GetServerUptimePercentage error: %w", err)
			s.loggingError(c, err, fmt.Sprintf("failed to get uptime percentage of server %s from %s to %s", id, startTime, endTime), zap.ErrorLevel)
			c.JSON(http.StatusInternalServerError, response.Response{
				Message: "Internal Server Error",
			})
			return
		}
		c.JSON(http.StatusOK, response.UptimeResponse{
			UptimePercentage: res,
		})
	}
}

func (s *serverHandler) GetServers() gin.HandlerFunc {
	return func(c *gin.Context) {
		serverName := c.Query("server_name")
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
		status := c.Query("status")
		if status != "" && status != model.ServerStatusPending && status != model.ServerStatusHealthy && status != model.ServerStatusUnhealthy && status != model.ServerStatusInactive && status != model.ServerStatusConfigurationError && status != model.ServerStatusNetworkError {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Invalid status",
			})
			return
		}
		sortBy := c.DefaultQuery("sort_by", "created_at")
		if sortBy != "server_name" && sortBy != "created_at" {
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
		servers, err := s.serverService.GetServers(c, serverName, status, sortBy, sortOrder, l, o)
		if err != nil {
			err = fmt.Errorf("ServerHandler.GetServers: %w", err)
			s.loggingError(c, err, "failed to get servers", zap.ErrorLevel)
			c.JSON(http.StatusInternalServerError, response.Response{
				Message: "Internal Server Error",
			})
			return
		}
		serversRes := make([]response.ServerInfoResponse, 0)
		for _, server := range servers {
			serversRes = append(serversRes, response.ServerInfoResponse{
				ID:                  server.ID,
				ServerName:          server.ServerName,
				Status:              server.Status,
				Ipv4:                server.Ipv4,
				Port:                server.Port,
				HealthEndpoint:      server.HealthEndpoint,
				HealthCheckInterval: server.HealthCheckInterval,
				CreatedAt:           server.CreatedAt,
				UpdatedAt:           server.UpdatedAt,
			})
		}
		c.JSON(http.StatusOK, serversRes)
	}
}

func (s *serverHandler) ExportServersToExcelFile() gin.HandlerFunc {
	return func(c *gin.Context) {
		serverName := c.Query("server_name")
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
		status := c.Query("status")
		if status != "" && status != model.ServerStatusPending && status != model.ServerStatusHealthy && status != model.ServerStatusUnhealthy && status != model.ServerStatusInactive && status != model.ServerStatusConfigurationError && status != model.ServerStatusNetworkError {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Invalid status",
			})
			return
		}
		sortBy := c.DefaultQuery("sort_by", "created_at")
		if sortBy != "server_name" && sortBy != "created_at" {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Invalid sort by",
			})
			return
		}
		sortOrder := c.DefaultQuery("sort_order", "desc")
		if sortOrder != "asc" && sortOrder != "desc" {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Invalid sort order",
			})
			return
		}
		servers, err := s.serverService.GetServers(c, serverName, status, sortBy, sortOrder, l, o)
		if err != nil {
			err = fmt.Errorf("ServerHandler.ExportServersToExcelFile: %w", err)
			s.loggingError(c, err, "failed to export servers", zap.ErrorLevel)
			c.JSON(http.StatusInternalServerError, response.Response{
				Message: "Internal server error",
			})
			return
		}
		file, err := s.generateExcelFile(servers)
		defer file.Close()
		if err != nil {
			err = fmt.Errorf("ServerHandler.ExportServersToExcelFile: %w", err)
			s.loggingError(c, err, "failed to export servers", zap.ErrorLevel)
			c.JSON(http.StatusInternalServerError, response.Response{
				Message: "Internal server error",
			})
			return
		}
		fileName := fmt.Sprintf("servers-%s.xlsx", time.Now().Format("2006-01-02T15:04:05"))
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
		if err = file.Write(c.Writer); err != nil {
			err = fmt.Errorf("ServerHandler.ExportServersToExcelFile: %w", err)
			s.loggingError(c, err, "failed to export servers", zap.ErrorLevel)
			c.JSON(http.StatusInternalServerError, response.Response{
				Message: "Internal server error",
			})
			return
		}
		c.Status(http.StatusOK)
	}
}

func (s *serverHandler) generateExcelFile(servers []model.Server) (*excelize.File, error) {
	f := excelize.NewFile()
	sheetName := "Servers"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, err
	}
	headers := []interface{}{"id", "server_name", "status", "ipv4", "port", "health_endpoint", "health_check_interval", "created_at", "updated_at"}
	headerStarCell := "A1"
	err = f.SetSheetRow(sheetName, headerStarCell, &headers)
	if err != nil {
		return nil, err
	}
	for i, server := range servers {
		rowData := []interface{}{
			server.ID,
			server.ServerName,
			server.Status,
			server.Ipv4,
			server.Port,
			server.HealthEndpoint,
			server.HealthCheckInterval,
			server.CreatedAt.Format("2006-01-02 15:04:05"),
			server.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		startCell := fmt.Sprintf("A%d", i+2)
		err = f.SetSheetRow(sheetName, startCell, &rowData)
		if err != nil {
			return nil, err
		}
	}
	f.SetActiveSheet(index)
	return f, nil
}

func (s *serverHandler) ReportAllServersHealthInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req request.ReportRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			var validatorError validator.ValidationErrors
			if errors.As(err, &validatorError) {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: s.formatValidationError(validatorError[0]),
				})
			} else {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Invalid request body",
				})
			}
			return
		}
		startTime, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Invalid start date",
			})
			return
		}
		endTime, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Invalid end date",
			})
			return
		}
		if endTime.Before(startTime) {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Invalid end date",
			})
			return
		}
		endTimeFinal := endTime.AddDate(0, 0, 1)
		err = s.serverService.ReportServersInformation(c, startTime, endTimeFinal, req.Email)
		if err != nil {
			err = fmt.Errorf("ServerHandler.ReportAllServersHealthInfo: %w", err)
			s.loggingError(c, err, "failed to reports servers", zap.ErrorLevel)
			c.JSON(http.StatusInternalServerError, response.Response{
				Message: "Internal server error",
			})
			return
		}
		c.JSON(http.StatusOK, response.Response{
			Message: "Report sent successfully",
		})
	}
}

func (s *serverHandler) CreateServer() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req request.ServerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			var validatorError validator.ValidationErrors
			if errors.As(err, &validatorError) {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: s.formatValidationError(validatorError[0]),
				})
			} else {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Invalid request body",
				})
			}
			return
		}
		newServer := model.Server{
			ServerName:          req.ServerName,
			Ipv4:                req.Ipv4,
			Port:                *req.Port,
			HealthEndpoint:      req.HealthEndpoint,
			HealthCheckInterval: *req.HealthCheckInterval,
		}
		res, err := s.serverService.CreateServer(c, newServer)
		if err != nil {
			switch {
			case errors.Is(err, apperrors.ErrServerNameAlreadyExists):
				c.JSON(http.StatusConflict, response.Response{
					Message: "Server name already exists",
				})
			default:
				err = fmt.Errorf("ServerHandler.CreateServer: %w", err)
				s.loggingError(c, err, "failed to create server", zap.ErrorLevel)
				c.JSON(http.StatusInternalServerError, response.Response{
					Message: "Internal server error",
				})
			}
			return
		}
		c.JSON(http.StatusCreated, response.ServerInfoResponse{
			ID:                  res.ID,
			ServerName:          res.ServerName,
			Status:              res.Status,
			Ipv4:                res.Ipv4,
			Port:                res.Port,
			HealthEndpoint:      res.HealthEndpoint,
			HealthCheckInterval: res.HealthCheckInterval,
			CreatedAt:           res.CreatedAt,
			UpdatedAt:           res.UpdatedAt,
		})

	}
}

func (s *serverHandler) ImportServersFromExcelFile() gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "Invalid request body",
			})
			return
		}
		ext := filepath.Ext(file.Filename)
		if ext != ".xlsx" && ext != ".xls" {
			c.JSON(http.StatusBadRequest, response.Response{
				Message: "File must be excel file",
			})
			return
		}
		importSheet := c.Query("sheet_name")

		validServers, invalidServers, err := s.extractServersFromExcelFile(file, importSheet)
		if err != nil {
			switch {
			case errors.Is(err, errEmptyFile):
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "File is empty",
				})
			case errors.Is(err, errSheetNotFound):
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Sheet not found",
				})
			case errors.Is(err, errMissingRequiredColumn):
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Missing required column",
				})
			default:
				err = fmt.Errorf("ServerHandler.ImportServersFromExcelFile: %w", err)
				s.loggingError(c, err, "failed to import server", zap.ErrorLevel)
				c.JSON(http.StatusInternalServerError, response.Response{
					Message: "Internal server error",
				})
			}
			return
		}

		importedServers, nonImportedServers, err := s.serverService.CreateServers(c, validServers)
		if err != nil {
			err = fmt.Errorf("ServerHandler.ImportServersFromExcelFile: %w", err)
			s.loggingError(c, err, "failed to import server", zap.ErrorLevel)
			c.JSON(http.StatusInternalServerError, response.Response{
				Message: "Internal server error",
			})
			return
		}
		var importedServerNames []string
		for _, importedServer := range importedServers {
			importedServerNames = append(importedServerNames, importedServer.ServerName)
		}
		for _, nonImportedServer := range nonImportedServers {
			invalidServers = append(invalidServers, nonImportedServer.ServerName)
		}
		c.JSON(http.StatusOK, response.ImportServerResponse{
			ImportedCount:   len(importedServerNames),
			ImportedServers: importedServerNames,
			FailedCount:     len(invalidServers),
			FailedServers:   invalidServers,
		})
	}
}

var errSheetNotFound = errors.New("sheet not found")
var errEmptyFile = errors.New("file is empty")
var errMissingRequiredColumn = errors.New("missing required column")

func (s *serverHandler) extractServersFromExcelFile(file *multipart.FileHeader, importSheet string) (validServers []model.Server, invalidServers []string, err error) {
	fileContent, err := file.Open()
	if err != nil {
		return
	}
	defer fileContent.Close()

	xlsx, err := excelize.OpenReader(fileContent)
	if err != nil {
		return
	}
	defer xlsx.Close()

	if importSheet == "" {
		importSheet = xlsx.GetSheetName(0)
	} else {
		index, _ := xlsx.GetSheetIndex(importSheet)
		if index == -1 {
			err = errSheetNotFound
			return
		}
	}

	rows, err := xlsx.GetRows(importSheet)
	if err != nil {
		return
	}
	if len(rows) < 2 {
		err = errEmptyFile
		return
	}

	columnMap := make(map[string]int)
	for i, cell := range rows[0] {
		columnMap[strings.ToLower(strings.TrimSpace(cell))] = i
	}
	requiredColumns := []string{"server_name", "ipv4", "port", "health_endpoint", "health_check_interval"}
	for _, requiredColumn := range requiredColumns {
		if _, ok := columnMap[requiredColumn]; !ok {
			err = errMissingRequiredColumn
			return
		}
	}

	for i, row := range rows {
		if i == 0 {
			continue
		}
		interval := row[columnMap["health_check_interval"]]
		intervalInt, e := strconv.Atoi(interval)
		if e != nil {
			invalidServers = append(invalidServers, row[columnMap["server_name"]])
			continue
		}
		p, e := strconv.Atoi(row[columnMap["port"]])
		if e != nil {
			invalidServers = append(invalidServers, row[columnMap["server_name"]])
			continue
		}
		req := request.ServerRequest{
			ServerName:          row[columnMap["server_name"]],
			Ipv4:                row[columnMap["ipv4"]],
			Port:                &p,
			HealthEndpoint:      row[columnMap["health_endpoint"]],
			HealthCheckInterval: &intervalInt,
		}
		if e = s.validator.Struct(req); e != nil {
			invalidServers = append(invalidServers, row[columnMap["server_name"]])
		} else {
			validServers = append(validServers, model.Server{
				ServerName:          req.ServerName,
				Ipv4:                req.Ipv4,
				Port:                *req.Port,
				HealthEndpoint:      req.HealthEndpoint,
				HealthCheckInterval: *req.HealthCheckInterval,
			})
		}
	}
	return
}

func (s *serverHandler) UpdateServer() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req request.UpdateServerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			var validatorError validator.ValidationErrors
			if errors.As(err, &validatorError) {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: s.formatValidationError(validatorError[0]),
				})
			} else {
				c.JSON(http.StatusBadRequest, response.Response{
					Message: "Invalid request body",
				})
			}
			return
		}
		id := c.Param("id")
		updatedData := model.Server{
			ID:                  id,
			ServerName:          req.ServerName,
			Ipv4:                req.Ipv4,
			Port:                *req.Port,
			HealthEndpoint:      req.HealthEndpoint,
			HealthCheckInterval: *req.HealthCheckInterval,
		}
		updatedServer, err := s.serverService.UpdateServer(c, updatedData)
		if err != nil {
			switch {
			case errors.Is(err, apperrors.ErrServerNotFound):
				c.JSON(http.StatusNotFound, response.Response{
					Message: "Server not found",
				})
			default:
				err = fmt.Errorf("ServerHandler.UpdateServer: %w", err)
				s.loggingError(c, err, fmt.Sprintf("failed to update server %s", id), zap.ErrorLevel)
				c.JSON(http.StatusInternalServerError, response.Response{
					Message: "Internal server error",
				})
			}
			return
		}
		c.JSON(http.StatusOK, response.ServerInfoResponse{
			ID:                  updatedServer.ID,
			ServerName:          updatedServer.ServerName,
			Status:              updatedServer.Status,
			Ipv4:                updatedServer.Ipv4,
			Port:                updatedServer.Port,
			HealthEndpoint:      updatedServer.HealthEndpoint,
			HealthCheckInterval: updatedServer.HealthCheckInterval,
			CreatedAt:           updatedServer.CreatedAt,
			UpdatedAt:           updatedServer.UpdatedAt,
		})
	}
}

func (s *serverHandler) DeleteServer() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		err := s.serverService.DeleteServer(c, id)
		if err != nil {
			err = fmt.Errorf("ServerHandler.DeleteServer: %w", err)
			s.loggingError(c, err, fmt.Sprintf("failed to delete server %s", id), zap.ErrorLevel)
			c.JSON(http.StatusInternalServerError, response.Response{
				Message: "Internal server error",
			})
			return
		}
		c.JSON(http.StatusOK, response.Response{
			Message: "Server deleted",
		})
	}
}

func (s *serverHandler) loggingError(c *gin.Context, err error, errDescription string, logLevel zapcore.Level) {
	var data []zapcore.Field
	data = append(data, zap.Error(err))
	data = append(data, zap.String("http_method", c.Request.Method))
	data = append(data, zap.String("http_path", c.Request.URL.Path))
	userId := c.GetHeader("X-User-Id")
	if userId != "" {
		data = append(data, zap.String("user_id", userId))
	}
	s.logger.Log(logLevel, errDescription, data...)
}

func NewServerHandler(logger *zap.Logger, serverService service.ServerService) ServerHandler {
	return &serverHandler{
		logger:        logger,
		serverService: serverService,
		validator:     validator.New(),
	}
}
