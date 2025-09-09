package repository

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRoundTripper struct {
	Response *http.Response
	Err      error
}

func (m *mockRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Response, nil
}

func newMockEsClient(statusCode int, body string, err error) (*elasticsearch.Client, error) {
	if err != nil {
		return elasticsearch.NewClient(elasticsearch.Config{
			Transport: &mockRoundTripper{Err: err},
		})
	}
	header := http.Header{}
	header.Set("Content-Type", "application/json")
	header.Set("X-Elastic-Product", "Elasticsearch")

	return elasticsearch.NewClient(elasticsearch.Config{
		Transport: &mockRoundTripper{
			Response: &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     header,
			},
		},
	})
}

func TestHealthCheckRepository_GetAllServersHealthInformation(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	successBody := `{
		"aggregations": {
			"avg_uptime_percentage": {
				"value": 95.5
			},
			"servers": {
				"buckets": [
					{
						"key": "server-1",
						"latest_check": { "hits": { "hits": [ { "_source": { "status": "healthy" } } ] } }
					},
					{
						"key": "server-2",
						"latest_check": { "hits": { "hits": [ { "_source": { "status": "unhealthy" } } ] } }
					},
					{
						"key": "server-3",
						"latest_check": { "hits": { "hits": [ { "_source": { "status": "healthy" } } ] } }
					},
					{
						"key": "server-4",
						"latest_check": { "hits": { "hits": [ { "_source": { "status": "configuration_error" } } ] } }
					},
					{
						"key": "server-5",
						"latest_check": { "hits": { "hits": [ { "_source": { "status": "network_error" } } ] } }
					},
					{
						"key": "server-6",
						"latest_check": { "hits": { "hits": [ { "_source": { "status": "some_other_status" } } ] } }
					}
				]
			}
		}
	}`

	esErrorBody := `{
		"error": {
			"type": "search_phase_exception",
			"reason": "bad query"
		}
	}`

	testCases := []struct {
		name           string
		mockStatusCode int
		mockBody       string
		mockErr        error
		output         ServersHealthInformation
		expectErr      bool
	}{
		{
			name:           "Success Should return aggregated server health information",
			mockStatusCode: http.StatusOK,
			mockBody:       successBody,
			output: ServersHealthInformation{
				TotalServersCnt:              6,
				HealthyServersCnt:            2,
				UnhealthyServersCnt:          1,
				InactiveServersCnt:           1,
				ConfigurationErrorServersCnt: 1,
				NetworkErrorServersCnt:       1,
				AverageUptimePercentage:      95.5,
			},
			expectErr: false,
		},
		{
			name:      "Error - Elasticsearch client transport error",
			mockErr:   errors.New("network connection failed"),
			output:    ServersHealthInformation{},
			expectErr: true,
		},
		{
			name:           "Error - Elasticsearch API returns an error",
			mockStatusCode: http.StatusBadRequest,
			mockBody:       esErrorBody,
			output:         ServersHealthInformation{},
			expectErr:      true,
		},
		{
			name:           "Error - Failed to decode Elasticsearch error response",
			mockStatusCode: http.StatusBadRequest,
			mockBody:       `{"error": "invalid json"`, // JSON không hợp lệ
			output:         ServersHealthInformation{},
			expectErr:      true,
		},
		{
			name:           "Error - Failed to decode success response",
			mockStatusCode: http.StatusOK,
			mockBody:       `{"aggregations": "invalid json"`, // JSON không hợp lệ
			output:         ServersHealthInformation{},
			expectErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockEsClient, err := newMockEsClient(tc.mockStatusCode, tc.mockBody, tc.mockErr)
			require.NoError(t, err)

			repo := NewHealthCheckRepository(nil, mockEsClient)

			got, err := repo.GetAllServersHealthInformation(context.Background(), startTime, endTime)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.output, got)
		})
	}
}

func TestHealthCheckRepository_GetServerUptimePercentage(t *testing.T) {
	serverID := "test-server-1"
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	successBody := `{
		"aggregations": {
			"uptime_percentage": {
				"value": 99.8
			}
		}
	}`

	esErrorBody := `{
		"error": {
			"type": "search_phase_exception",
			"reason": "bad query"
		}
	}`

	testCases := []struct {
		name           string
		mockStatusCode int
		mockBody       string
		mockErr        error
		output         float64
		expectErr      bool
	}{
		{
			name:           "Success Should return server uptime percentage",
			mockStatusCode: http.StatusOK,
			mockBody:       successBody,
			output:         99.8,
			expectErr:      false,
		},
		{
			name:           "Success Should return 0 if value is null (not present in ES response)",
			mockStatusCode: http.StatusOK,
			mockBody:       `{"aggregations": {"uptime_percentage": {"value": null}}}`,
			output:         0,
			expectErr:      false,
		},
		{
			name:      "Error Elasticsearch client transport error",
			mockErr:   errors.New("network connection failed"),
			output:    0,
			expectErr: true,
		},
		{
			name:           "Error Elasticsearch API returns an error",
			mockStatusCode: http.StatusBadRequest,
			mockBody:       esErrorBody,
			output:         0,
			expectErr:      true,
		},
		{
			name:           "Error Failed to decode Elasticsearch error response",
			mockStatusCode: http.StatusBadRequest,
			mockBody:       `{"error": "invalid json"`,
			output:         0,
			expectErr:      true,
		},
		{
			name:           "Error Failed to decode success response",
			mockStatusCode: http.StatusOK,
			mockBody:       `{"aggregations": "invalid json"`,
			output:         0,
			expectErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockEsClient, err := newMockEsClient(tc.mockStatusCode, tc.mockBody, tc.mockErr)
			require.NoError(t, err)

			repo := NewHealthCheckRepository(nil, mockEsClient)

			got, err := repo.GetServerUptimePercentage(context.Background(), serverID, startTime, endTime)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.output, got)
		})
	}
}
