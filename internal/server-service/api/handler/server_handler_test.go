package handler

import (
	"VCS_SMS_Microservice/internal/server-service/api/dto/request"
	"VCS_SMS_Microservice/internal/server-service/api/dto/response"
	apperrors "VCS_SMS_Microservice/internal/server-service/errors"
	mockservice "VCS_SMS_Microservice/internal/server-service/mocks/service"
	"VCS_SMS_Microservice/internal/server-service/model"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func setupTestContext(t *testing.T, method, url string, body io.Reader) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	c.Request = req
	return w, c
}

func TestServerHandler_CreateServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	port := 8080
	interval := 30

	serverReq := request.ServerRequest{
		ServerName:          "TestServer",
		Ipv4:                "127.0.0.1",
		Port:                &port,
		HealthEndpoint:      "/health",
		HealthCheckInterval: &interval,
	}
	serverModel := model.Server{
		ServerName:          serverReq.ServerName,
		Ipv4:                serverReq.Ipv4,
		Port:                *serverReq.Port,
		HealthEndpoint:      serverReq.HealthEndpoint,
		HealthCheckInterval: *serverReq.HealthCheckInterval,
	}
	createdServer := model.Server{
		ID:                  "uuid-123",
		ServerName:          "TestServer",
		Status:              "pending",
		Ipv4:                "127.0.0.1",
		Port:                8080,
		HealthEndpoint:      "/health",
		HealthCheckInterval: 30,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	testCases := []struct {
		name           string
		body           interface{}
		setupMocks     func(mockService *mockservice.MockServerService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success Server Created",
			body: serverReq,
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().CreateServer(gomock.Any(), serverModel).Return(createdServer, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `"id":"uuid-123"`,
		},
		{
			name:           "Error Invalid JSON body",
			body:           `{"server_name": "Test"`,
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Invalid request body"`,
		},
		{
			name:           "Error Validation Failed (required field)",
			body:           request.ServerRequest{Ipv4: "127.0.0.1"}, // Thiếu ServerName
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"The ServerName field is required"`,
		},
		{
			name: "Error Server Name Already Exists",
			body: serverReq,
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().CreateServer(gomock.Any(), serverModel).Return(model.Server{}, apperrors.ErrServerNameAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   `"message":"Server name already exists"`,
		},
		{
			name: "Error Internal Server Error",
			body: serverReq,
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().CreateServer(gomock.Any(), serverModel).Return(model.Server{}, errors.New("unexpected db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `"message":"Internal server error"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockService := mockservice.NewMockServerService(ctrl)
			tc.setupMocks(mockService)

			handler := NewServerHandler(zap.NewNop(), mockService)

			var reqBody io.Reader
			if bodyStr, ok := tc.body.(string); ok {
				reqBody = strings.NewReader(bodyStr)
			} else {
				jsonBody, _ := json.Marshal(tc.body)
				reqBody = bytes.NewReader(jsonBody)
			}

			w, c := setupTestContext(t, http.MethodPost, "/servers", reqBody)
			c.Request.Header.Set("Content-Type", "application/json")

			handler.CreateServer()(c)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tc.expectedBody)
		})
	}
}

func TestServerHandler_GetServers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	serversList := []model.Server{
		{ID: "1", ServerName: "ServerA"},
		{ID: "2", ServerName: "ServerB"},
	}

	testCases := []struct {
		name           string
		url            string
		setupMocks     func(mockService *mockservice.MockServerService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success Get servers with default params",
			url:  "/servers",
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().GetServers(gomock.Any(), "", "", "created_at", "asc", 10, 0).Return(serversList, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"1","server_name":"ServerA"`,
		},
		{
			name: "Success - Get servers with all params",
			url:  "/servers?server_name=A&status=healthy&sort_by=server_name&sort_order=desc&limit=5&offset=1",
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().GetServers(gomock.Any(), "A", "healthy", "server_name", "desc", 5, 1).Return([]model.Server{serversList[0]}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"1","server_name":"ServerA"`,
		},
		{
			name:           "Error Invalid offset",
			url:            "/servers?offset=abc",
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Offset must be an integer"`,
		},
		{
			name:           "Error Invalid limit",
			url:            "/servers?limit=xyz",
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Limit must be an integer"`,
		},
		{
			name:           "Error Invalid status",
			url:            "/servers?status=invalid_status",
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Invalid status"`,
		},
		{
			name:           "Error Invalid sort_by",
			url:            "/servers?sort_by=invalid_column",
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Invalid sort by"`,
		},
		{
			name:           "Error Invalid sort_order",
			url:            "/servers?sort_order=invalid_order",
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Invalid sort order"`,
		},
		{
			name: "Error - Service Error",
			url:  "/servers",
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().GetServers(gomock.Any(), "", "", "created_at", "asc", 10, 0).Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `"message":"Internal Server Error"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockService := mockservice.NewMockServerService(ctrl)
			tc.setupMocks(mockService)

			handler := NewServerHandler(zap.NewNop(), mockService)

			w, c := setupTestContext(t, http.MethodGet, tc.url, nil)
			handler.GetServers()(c)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tc.expectedBody)
		})
	}
}

func TestServerHandler_DeleteServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serverID := "server-id-123"

	testCases := []struct {
		name           string
		serverID       string
		setupMocks     func(mockService *mockservice.MockServerService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:     "Success Server Deleted",
			serverID: serverID,
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().DeleteServer(gomock.Any(), serverID).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"message":"Server deleted"`,
		},
		{
			name:     "Error Service Fails to Delete",
			serverID: serverID,
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().DeleteServer(gomock.Any(), serverID).Return(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `"message":"Internal server error"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockService := mockservice.NewMockServerService(ctrl)
			tc.setupMocks(mockService)

			handler := NewServerHandler(zap.NewNop(), mockService)

			w, c := setupTestContext(t, http.MethodDelete, "/servers/"+tc.serverID, nil)
			c.Params = gin.Params{gin.Param{Key: "id", Value: tc.serverID}}

			handler.DeleteServer()(c)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tc.expectedBody)
		})
	}
}

func TestServerHandler_GetServerUptimePercentage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serverID := "server-uuid-123"

	validStartDate := "2025-09-01"
	validEndDate := "2025-09-07"

	expectedStartTime, _ := time.Parse("2006-01-02", validStartDate)
	expectedEndTime, _ := time.Parse("2006-01-02", validEndDate)
	expectedEndTimeFinal := expectedEndTime.AddDate(0, 0, 1)

	testCases := []struct {
		name           string
		url            string
		setupMocks     func(mockService *mockservice.MockServerService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success - Get Uptime Percentage",
			url:  fmt.Sprintf("/servers/%s/uptime?start_date=%s&end_date=%s", serverID, validStartDate, validEndDate),
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().
					GetServerUptimePercentage(gomock.Any(), serverID, expectedStartTime, expectedEndTimeFinal).
					Return(99.8, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"uptime_percentage":99.8}`,
		},
		{
			name:           "Error - Invalid Start Date Format",
			url:            fmt.Sprintf("/servers/%s/uptime?start_date=01-09-2025&end_date=%s", serverID, validEndDate),
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Invalid start date"`,
		},
		{
			name:           "Error - Invalid End Date Format",
			url:            fmt.Sprintf("/servers/%s/uptime?start_date=%s&end_date=not-a-date", serverID, validStartDate),
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Invalid end date"`,
		},
		{
			name:           "Error - End Date Before Start Date",
			url:            fmt.Sprintf("/servers/%s/uptime?start_date=%s&end_date=2025-08-31", serverID, validStartDate),
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Invalid end date"`,
		},
		{
			name: "Error - Service Returns an Error",
			url:  fmt.Sprintf("/servers/%s/uptime?start_date=%s&end_date=%s", serverID, validStartDate, validEndDate),
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().
					GetServerUptimePercentage(gomock.Any(), serverID, expectedStartTime, expectedEndTimeFinal).
					Return(0.0, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `"message":"Internal Server Error"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockService := mockservice.NewMockServerService(ctrl)
			tc.setupMocks(mockService)

			handler := NewServerHandler(zap.NewNop(), mockService)

			w, c := setupTestContext(t, http.MethodGet, tc.url, nil)

			c.Params = gin.Params{
				gin.Param{Key: "id", Value: serverID},
			}

			handler.GetServerUptimePercentage()(c)

			assert.Equal(t, tc.expectedStatus, w.Code)

			assert.Contains(t, w.Body.String(), tc.expectedBody)
		})
	}
}

func TestServerHandler_ExportServersToExcelFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockServers := []model.Server{
		{
			ID:                  "uuid-1",
			ServerName:          "WebApp-01",
			Status:              "healthy",
			Ipv4:                "192.168.1.10",
			Port:                80,
			HealthEndpoint:      "/health",
			HealthCheckInterval: 60,
			CreatedAt:           time.Date(2025, 9, 8, 10, 0, 0, 0, time.UTC),
			UpdatedAt:           time.Date(2025, 9, 8, 11, 0, 0, 0, time.UTC),
		},
		{
			ID:                  "uuid-2",
			ServerName:          "Database-01",
			Status:              "unhealthy",
			Ipv4:                "192.168.1.11",
			Port:                5432,
			HealthEndpoint:      "/status",
			HealthCheckInterval: 30,
			CreatedAt:           time.Date(2025, 9, 7, 12, 0, 0, 0, time.UTC),
			UpdatedAt:           time.Date(2025, 9, 8, 13, 0, 0, 0, time.UTC),
		},
	}

	testCases := []struct {
		name               string
		url                string
		setupMocks         func(mockService *mockservice.MockServerService)
		expectedStatus     int
		expectedHeaders    map[string]string
		expectedBody       string
		verifyExcelContent func(t *testing.T, body *bytes.Buffer, servers []model.Server)
	}{
		{
			name: "Success Export servers to Excel",
			url:  "/servers/export?status=healthy",
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().GetServers(gomock.Any(), "", "healthy", "created_at", "desc", 10, 0).Return(mockServers, nil)
			},
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Content-Type": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			},
			verifyExcelContent: func(t *testing.T, body *bytes.Buffer, servers []model.Server) {
				f, err := excelize.OpenReader(body)
				assert.NoError(t, err)

				rows, err := f.GetRows("Servers")
				assert.NoError(t, err)
				assert.Len(t, rows, 3)

				expectedHeaders := []string{"id", "server_name", "status", "ipv4", "port", "health_endpoint", "health_check_interval", "created_at", "updated_at"}
				assert.Equal(t, expectedHeaders, rows[0])

				firstServer := servers[0]
				expectedFirstRow := []string{
					firstServer.ID,
					firstServer.ServerName,
					firstServer.Status,
					firstServer.Ipv4,
					fmt.Sprintf("%d", firstServer.Port),
					firstServer.HealthEndpoint,
					fmt.Sprintf("%d", firstServer.HealthCheckInterval),
					firstServer.CreatedAt.Format("2006-01-02 15:04:05"),
					firstServer.UpdatedAt.Format("2006-01-02 15:04:05"),
				}
				assert.Equal(t, expectedFirstRow, rows[1])
			},
		},
		{
			name: "Error Invalid Query Parameter (status)",
			url:  "/servers/export?status=invalid",
			setupMocks: func(mockService *mockservice.MockServerService) {
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Invalid status"`,
		},
		{
			name:           "Error Invalid Query Parameter (limit)",
			url:            "/servers/export?limit=abc",
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Limit must be an integer"`,
		},
		{
			name: "Error Service Fails to Get Servers",
			url:  "/servers/export",
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().GetServers(gomock.Any(), "", "", "created_at", "desc", 10, 0).Return(nil, errors.New("database is down"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `"message":"Internal server error"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockService := mockservice.NewMockServerService(ctrl)
			tc.setupMocks(mockService)

			handler := NewServerHandler(zap.NewNop(), mockService)

			w, c := setupTestContext(t, http.MethodGet, tc.url, nil)
			handler.ExportServersToExcelFile()(c)

			assert.Equal(t, tc.expectedStatus, w.Code)
			for key, value := range tc.expectedHeaders {
				assert.Equal(t, value, w.Header().Get(key))
			}
			if tc.expectedStatus == http.StatusOK {
				contentDisposition := w.Header().Get("Content-Disposition")
				assert.True(t, strings.HasPrefix(contentDisposition, `attachment; filename="servers-`))
				assert.True(t, strings.HasSuffix(contentDisposition, `.xlsx"`))
			}
			if tc.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tc.expectedBody)
			}
			if tc.verifyExcelContent != nil {
				tc.verifyExcelContent(t, w.Body, mockServers)
			}
		})
	}
}

func TestServerHandler_ReportAllServersHealthInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validReq := request.ReportRequest{
		Email:     "admin@example.com",
		StartDate: "2025-09-01",
		EndDate:   "2025-09-07",
	}

	startTime, _ := time.Parse("2006-01-02", validReq.StartDate)
	endTime, _ := time.Parse("2006-01-02", validReq.EndDate)
	endTimeFinal := endTime.AddDate(0, 0, 1)

	testCases := []struct {
		name           string
		body           interface{}
		setupMocks     func(mockService *mockservice.MockServerService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success Report sent",
			body: validReq,
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().
					ReportServersInformation(gomock.Any(), startTime, endTimeFinal, validReq.Email).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"message":"Report sent successfully"`,
		},
		{
			name:           "Error Malformed JSON",
			body:           `{"email": "admin@example.com", "start_date": "2025-09-01"`, // Missing closing brace
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Invalid request body"`,
		},
		{
			name:           "Error Validation Failed (Missing required email)",
			body:           request.ReportRequest{StartDate: "2025-09-01", EndDate: "2025-09-07"},
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"The Email field is required"`,
		},
		{
			name:           "Error Validation Failed (Invalid email format)",
			body:           request.ReportRequest{Email: "not-an-email", StartDate: "2025-09-01", EndDate: "2025-09-07"},
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"The Email field is not a valid email"`,
		},
		{
			name:           "Error Invalid Start Date Format",
			body:           request.ReportRequest{Email: "admin@example.com", StartDate: "01-09-2025", EndDate: "2025-09-07"},
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"The StartDate field is not a valid datetime, use YYYY-MM-DD format"`,
		},
		{
			name:           "Error - End Date Before Start Date",
			body:           request.ReportRequest{Email: "admin@example.com", StartDate: "2025-09-07", EndDate: "2025-09-01"},
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Invalid end date"`,
		},
		{
			name: "Error - Service Fails to Send Report",
			body: validReq,
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().
					ReportServersInformation(gomock.Any(), startTime, endTimeFinal, validReq.Email).
					Return(errors.New("SMTP server connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `"message":"Internal server error"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mockservice.NewMockServerService(ctrl)
			tc.setupMocks(mockService)

			handler := NewServerHandler(zap.NewNop(), mockService)

			var reqBody io.Reader
			if bodyStr, ok := tc.body.(string); ok {
				reqBody = bytes.NewBufferString(bodyStr)
			} else {
				jsonBytes, err := json.Marshal(tc.body)
				assert.NoError(t, err)
				reqBody = bytes.NewReader(jsonBytes)
			}

			w, c := setupTestContext(t, http.MethodPost, "/servers/report", reqBody)
			c.Request.Header.Set("Content-Type", "application/json")

			handler.ReportAllServersHealthInfo()(c)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tc.expectedBody)
		})
	}
}

func createTestExcelFile(t *testing.T, sheetName string, headers []string, data [][]interface{}) *bytes.Buffer {
	f := excelize.NewFile()
	index, _ := f.NewSheet(sheetName)

	// Set headers
	if len(headers) > 0 {
		err := f.SetSheetRow(sheetName, "A1", &headers)
		assert.NoError(t, err)
	}

	// Set data rows
	for i, rowData := range data {
		cell := fmt.Sprintf("A%d", i+2)
		err := f.SetSheetRow(sheetName, cell, &rowData)
		assert.NoError(t, err)
	}

	f.SetActiveSheet(index)

	// Save the file to a buffer
	buf, err := f.WriteToBuffer()
	assert.NoError(t, err)
	return buf
}

// Helper to create an HTTP request with a multipart/form-data body
func createMultipartRequest(t *testing.T, url, fieldName, fileName string, fileContent *bytes.Buffer) *http.Request {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	assert.NoError(t, err)
	_, err = io.Copy(part, fileContent)
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, url, body)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestServerHandler_ImportServersFromExcelFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	defaultSheet := "Sheet1"
	validHeaders := []string{"server_name", "ipv4", "port", "health_endpoint", "health_check_interval"}

	validData := [][]interface{}{
		{"WebApp-01", "192.168.1.10", "80", "/health", "60"},
		{"DB-01", "192.168.1.11", "5432", "/status", "30"},
	}

	expectedValidServers := []model.Server{
		{ServerName: "WebApp-01", Ipv4: "192.168.1.10", Port: 80, HealthEndpoint: "/health", HealthCheckInterval: 60},
		{ServerName: "DB-01", Ipv4: "192.168.1.11", Port: 5432, HealthEndpoint: "/status", HealthCheckInterval: 30},
	}

	testCases := []struct {
		name                string
		fileName            string
		sheetQueryParam     string
		excelFileContent    *bytes.Buffer
		setupMocks          func(mockService *mockservice.MockServerService)
		expectedStatus      int
		expectedBodyContain string
	}{
		{
			name:             "Success Import all servers",
			fileName:         "servers.xlsx",
			excelFileContent: createTestExcelFile(t, defaultSheet, validHeaders, validData),
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().ImportServers(gomock.Any(), expectedValidServers).Return(expectedValidServers, []model.Server{}, nil)
			},
			expectedStatus:      http.StatusOK,
			expectedBodyContain: `"imported_count":2`,
		},
		{
			name:                "Error No file provided",
			fileName:            "",
			excelFileContent:    bytes.NewBuffer(nil),
			setupMocks:          func(mockService *mockservice.MockServerService) {},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyContain: `"message":"Invalid request body"`,
		},
		{
			name:                "Error Wrong file extension",
			fileName:            "servers.txt",
			excelFileContent:    bytes.NewBufferString("this is a text file"),
			setupMocks:          func(mockService *mockservice.MockServerService) {},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyContain: `"message":"File must be excel file"`,
		},
		{
			name:                "Error Empty Excel file (only header)",
			fileName:            "empty.xlsx",
			excelFileContent:    createTestExcelFile(t, defaultSheet, validHeaders, [][]interface{}{}),
			setupMocks:          func(mockService *mockservice.MockServerService) {},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyContain: `"message":"File is empty"`,
		},
		{
			name:                "Error Sheet not found",
			fileName:            "servers.xlsx",
			sheetQueryParam:     "NonExistentSheet",
			excelFileContent:    createTestExcelFile(t, defaultSheet, validHeaders, validData),
			setupMocks:          func(mockService *mockservice.MockServerService) {},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyContain: `"message":"Sheet not found"`,
		},
		{
			name:                "Error Missing required column",
			fileName:            "missing_column.xlsx",
			excelFileContent:    createTestExcelFile(t, defaultSheet, []string{"server_name", "port"}, validData),
			setupMocks:          func(mockService *mockservice.MockServerService) {},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyContain: `"message":"Missing required column"`,
		},
		{
			name:     "Partial Success Some rows invalid, some imported",
			fileName: "mixed_data.xlsx",
			excelFileContent: createTestExcelFile(t, defaultSheet, validHeaders, [][]interface{}{
				{"ValidServer", "10.0.0.1", "8080", "/health", "15"},
				{"InvalidPort", "10.0.0.2", "not-a-port", "/health", "15"},
				{"InvalidIP", "not-an-ip", "80", "/health", "15"},
			}),
			setupMocks: func(mockService *mockservice.MockServerService) {
				validServer := []model.Server{{ServerName: "ValidServer", Ipv4: "10.0.0.1", Port: 8080, HealthEndpoint: "/health", HealthCheckInterval: 15}}
				mockService.EXPECT().ImportServers(gomock.Any(), validServer).Return(validServer, []model.Server{}, nil)
			},
			expectedStatus:      http.StatusOK,
			expectedBodyContain: `"imported_count":1,"imported_servers":["ValidServer"],"failed_count":2,"failed_servers":["InvalidPort","InvalidIP"]`,
		},
		{
			name:             "Error Service Fails",
			fileName:         "servers.xlsx",
			excelFileContent: createTestExcelFile(t, defaultSheet, validHeaders, validData),
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().ImportServers(gomock.Any(), expectedValidServers).Return(nil, nil, errors.New("database transaction failed"))
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyContain: `"message":"Internal server error"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mockservice.NewMockServerService(ctrl)
			tc.setupMocks(mockService)

			handler := NewServerHandler(zap.NewNop(), mockService)

			url := "/servers/import"
			if tc.sheetQueryParam != "" {
				url = url + "?sheet_name=" + tc.sheetQueryParam
			}

			var req *http.Request
			if tc.fileName == "" {
				req, _ = http.NewRequest(http.MethodPost, url, nil)
			} else {
				req = createMultipartRequest(t, url, "file", tc.fileName, tc.excelFileContent)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler.ImportServersFromExcelFile()(c)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var resp response.ImportServerResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Contains(t, w.Body.String(), tc.expectedBodyContain)
			} else {
				assert.Contains(t, w.Body.String(), tc.expectedBodyContain)
			}
		})
	}
}

func TestServerHandler_UpdateServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serverID := "server-uuid-123"
	port := 8081
	interval := 45

	validReq := request.UpdateServerRequest{
		ServerName:          "Updated-WebApp",
		Ipv4:                "192.168.1.100",
		Port:                &port,
		HealthEndpoint:      "/new-health",
		HealthCheckInterval: &interval,
	}

	expectedModel := model.Server{
		ID:                  serverID,
		ServerName:          validReq.ServerName,
		Ipv4:                validReq.Ipv4,
		Port:                *validReq.Port,
		HealthEndpoint:      validReq.HealthEndpoint,
		HealthCheckInterval: *validReq.HealthCheckInterval,
	}

	updatedServerResponse := model.Server{
		ID:                  serverID,
		ServerName:          "Updated-WebApp",
		Status:              "pending",
		Ipv4:                "192.168.1.100",
		Port:                8081,
		HealthEndpoint:      "/new-health",
		HealthCheckInterval: 45,
		CreatedAt:           time.Now().Add(-time.Hour),
		UpdatedAt:           time.Now(),
	}

	testCases := []struct {
		name           string
		body           interface{}
		setupMocks     func(mockService *mockservice.MockServerService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success - Server Updated",
			body: validReq,
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().
					UpdateServer(gomock.Any(), expectedModel).
					Return(updatedServerResponse, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"id":"server-uuid-123","server_name":"Updated-WebApp"`,
		},
		{
			name:           "Error - Malformed JSON",
			body:           `{"server_name": "Incomplete}`,
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Invalid request body"`,
		},
		{
			name: "Error - Validation Failed (Invalid IPv4)",
			body: request.UpdateServerRequest{
				ServerName:          "Test",
				Ipv4:                "not-an-ip", // Invalid value
				Port:                &port,
				HealthEndpoint:      "/health",
				HealthCheckInterval: &interval,
			},
			setupMocks:     func(mockService *mockservice.MockServerService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"The Ipv4 field is not a valid ipv4"`,
		},
		{
			name: "Error - Server Not Found",
			body: validReq,
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().
					UpdateServer(gomock.Any(), expectedModel).
					Return(model.Server{}, apperrors.ErrServerNotFound) // Specific error
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `"message":"Server not found"`,
		},
		{
			name: "Error - Internal Server Error on Update",
			body: validReq,
			setupMocks: func(mockService *mockservice.MockServerService) {
				mockService.EXPECT().
					UpdateServer(gomock.Any(), expectedModel).
					Return(model.Server{}, errors.New("unexpected database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `"message":"Internal server error"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mockservice.NewMockServerService(ctrl)
			tc.setupMocks(mockService)

			handler := NewServerHandler(zap.NewNop(), mockService)

			var reqBody io.Reader
			if bodyStr, ok := tc.body.(string); ok {
				reqBody = bytes.NewBufferString(bodyStr)
			} else {
				jsonBytes, err := json.Marshal(tc.body)
				assert.NoError(t, err)
				reqBody = bytes.NewReader(jsonBytes)
			}

			w, c := setupTestContext(t, http.MethodPut, "/servers/"+serverID, reqBody)
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{
				gin.Param{Key: "id", Value: serverID},
			}

			handler.UpdateServer()(c)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tc.expectedBody)
		})
	}
}
