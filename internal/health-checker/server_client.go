package health_checker

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"syscall"
	"time"
)

type ServerClient interface {
	GetServerHealthCheck(ctx context.Context, ipv4 string, port int, healthEndpoint string) (HealthCheckResponse, error)
}

type serverClient struct {
	client         *http.Client
	maxRetries     int
	initialBackoff time.Duration
}

// GetServerHealthCheck return error when failed to create *http.Request, error received when execute request is in HealthCheckResponse.Error
func (s serverClient) GetServerHealthCheck(ctx context.Context, ipv4 string, port int, healthEndpoint string) (HealthCheckResponse, error) {
	if !strings.HasPrefix(healthEndpoint, "/") {
		healthEndpoint = "/" + healthEndpoint
	}
	requestUrl := fmt.Sprintf("http://%s:%d%s", ipv4, port, healthEndpoint)
	backoff := s.initialBackoff
	var err error
	for attempt := 1; attempt <= s.maxRetries; attempt++ {
		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, requestUrl, nil)
		if err != nil {
			return HealthCheckResponse{}, fmt.Errorf("ServerClient.GetServerHealthCheck creating request: %w", err)
		}
		var resp *http.Response
		resp, err = s.client.Do(req)
		if err != nil {
			if errors.Is(err, syscall.ECONNREFUSED) {
				break
			} else {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
		}
		res := HealthCheckResponse{
			StatusCode: resp.StatusCode,
			Attempts:   attempt,
			Timestamp:  time.Now(),
		}
		resp.Body.Close()
		return res, nil
	}
	return HealthCheckResponse{
		Error:     err,
		Attempts:  s.maxRetries,
		Timestamp: time.Now(),
	}, nil
}

type HealthCheckResponse struct {
	StatusCode int
	Error      error
	Attempts   int
	Timestamp  time.Time
}

func NewManagedServerClient(maxRetries int, requestTimeout time.Duration, initialBackoff time.Duration) ServerClient {
	return &serverClient{
		client: &http.Client{
			Timeout: requestTimeout,
		},
		maxRetries:     maxRetries,
		initialBackoff: initialBackoff,
	}
}
