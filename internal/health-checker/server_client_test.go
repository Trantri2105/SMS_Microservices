package health_checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerClient_GetServerHealthChecks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	mockURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	host := mockURL.Hostname()
	port, err := strconv.Atoi(mockURL.Port())
	require.NoError(t, err)
	client := NewServerClient(3, 5*time.Second, 1*time.Second)
	testCases := []struct {
		name           string
		host           string
		port           int
		healthEndpoint string
		expectedOutput HealthCheckResponse
		expectError    bool
	}{
		{
			name:           "Healthcheck with valid information",
			host:           host,
			port:           port,
			healthEndpoint: "/health",
			expectedOutput: HealthCheckResponse{
				StatusCode: http.StatusOK,
				Error:      nil,
			},
		},
		{
			name:           "Healthcheck with valid information (health endpoint without / at the beginning)",
			host:           host,
			port:           port,
			healthEndpoint: "health",
			expectedOutput: HealthCheckResponse{
				StatusCode: http.StatusOK,
				Error:      nil,
			},
		},
		{
			name:           "Healthcheck with invalid health endpoint",
			host:           host,
			port:           port,
			healthEndpoint: "/info",
			expectedOutput: HealthCheckResponse{
				StatusCode: http.StatusNotFound,
			},
		},
		{
			name:           "Healthcheck with invalid url",
			host:           "",
			port:           -1,
			healthEndpoint: "//info",
			expectError:    true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			res, e := client.GetServerHealthCheck(ctx, tc.host, tc.port, tc.healthEndpoint)
			cancel()
			if tc.expectError {
				assert.Error(t, e)
			} else {
				assert.Equal(t, tc.expectedOutput.StatusCode, res.StatusCode)
				assert.ErrorIs(t, res.Error, tc.expectedOutput.Error)
			}
		})
	}
}
