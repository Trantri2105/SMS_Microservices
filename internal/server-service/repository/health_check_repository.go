package repository

import (
	apperrors "VCS_SMS_Microservice/internal/server-service/errors"
	"VCS_SMS_Microservice/internal/server-service/model"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v9"
	"gorm.io/gorm"
	"time"
)

type ServersHealthInformation struct {
	TotalServersCnt              int
	HealthyServersCnt            int
	UnhealthyServersCnt          int
	InactiveServersCnt           int
	ConfigurationErrorServersCnt int
	NetworkErrorServersCnt       int
	AverageUptimePercentage      float64
}

type HealthCheckRepository interface {
	GetServerUptimePercentage(ctx context.Context, serverID string, startTime time.Time, endTime time.Time) (float64, error)
	GetAllServersHealthInformation(ctx context.Context, startTime time.Time, endTime time.Time) (ServersHealthInformation, error)
}

const esHealthCheckIndexName = "health_checks"

type healthCheckRepository struct {
	es *elasticsearch.Client
}

type esErrorResponse struct {
	Error struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
	}
}

type esServersHealthResponse struct {
	Aggregations struct {
		AvgUptimePercentage struct {
			Value float64 `json:"value"`
		} `json:"avg_uptime_percentage"`
		Servers struct {
			Buckets []struct {
				Key         string `json:"key"`
				LatestCheck struct {
					Hits struct {
						Hits []struct {
							Source struct {
								Status string `json:"status"`
							} `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"latest_check"`
			} `json:"buckets"`
		} `json:"servers"`
	} `json:"aggregations"`
}

func (h *healthCheckRepository) GetAllServersHealthInformation(ctx context.Context, startTime time.Time, endTime time.Time) (ServersHealthInformation, error) {
	query := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"range": map[string]interface{}{
				"timestamp": map[string]interface{}{
					"gte": startTime,
					"lt":  endTime,
				},
			},
		},
		"aggs": map[string]interface{}{
			"avg_uptime_percentage": map[string]interface{}{
				"weighted_avg": map[string]interface{}{
					"value": map[string]interface{}{
						"field": "status_numeric",
					},
					"weight": map[string]interface{}{
						"field": "interval_since_last_health_check_ms",
					},
				},
			},
			"servers": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "server_id",
					"size":  20000,
				},
				"aggs": map[string]interface{}{
					"latest_check": map[string]interface{}{
						"top_hits": map[string]interface{}{
							"size": 1,
							"sort": []map[string]interface{}{
								{
									"timestamp": map[string]interface{}{
										"order": "desc",
									},
								},
							},
							"_source": map[string]interface{}{
								"includes": "status",
							},
						},
					},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return ServersHealthInformation{}, fmt.Errorf("HealthCheckRepo.GetAllServersHealthInformation encode query: %w", err)
	}
	res, err := h.es.Search(
		h.es.Search.WithContext(ctx),
		h.es.Search.WithIndex(esHealthCheckIndexName),
		h.es.Search.WithBody(&buf))
	if err != nil {
		return ServersHealthInformation{}, fmt.Errorf("HealthCheckRepo.GetAllServersHealthInformation: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e esErrorResponse
		if err = json.NewDecoder(res.Body).Decode(&e); err != nil {
			return ServersHealthInformation{}, fmt.Errorf("HealthCheckRepo.GetAllServersHealthInformation decode err response: %w", err)
		}
		return ServersHealthInformation{}, fmt.Errorf("HealthCheckRepo.GetAllServersHealthInformation: %w", apperrors.NewElasticSearchError(res.StatusCode, e.Error.Type, e.Error.Reason))
	}

	var serversHealthRes esServersHealthResponse
	if err = json.NewDecoder(res.Body).Decode(&serversHealthRes); err != nil {
		return ServersHealthInformation{}, fmt.Errorf("HealthCheckRepo.GetAllServersHealthInformation decode response body: %w", err)
	}
	serversHealth := ServersHealthInformation{
		TotalServersCnt:         len(serversHealthRes.Aggregations.Servers.Buckets),
		AverageUptimePercentage: serversHealthRes.Aggregations.AvgUptimePercentage.Value,
	}
	for _, bucket := range serversHealthRes.Aggregations.Servers.Buckets {
		status := bucket.LatestCheck.Hits.Hits[0].Source.Status
		if status == model.ServerStatusHealthy {
			serversHealth.HealthyServersCnt += 1
		} else if status == model.ServerStatusUnhealthy {
			serversHealth.UnhealthyServersCnt += 1
		} else if status == model.ServerStatusConfigurationError {
			serversHealth.ConfigurationErrorServersCnt += 1
		} else if status == model.ServerStatusNetworkError {
			serversHealth.NetworkErrorServersCnt += 1
		} else {
			serversHealth.InactiveServersCnt += 1
		}
	}
	return serversHealth, nil
}

type esUptimePercentageResponse struct {
	Aggregations struct {
		UptimePercentage struct {
			Value float64 `json:"value"`
		} `json:"uptime_percentage"`
	} `json:"aggregations"`
}

func (h *healthCheckRepository) GetServerUptimePercentage(ctx context.Context, serverID string, startTime time.Time, endTime time.Time) (float64, error) {
	query := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"server_id": serverID,
						},
					},
					{
						"range": map[string]interface{}{
							"timestamp": map[string]interface{}{
								"gte": startTime,
								"lt":  endTime,
							},
						},
					},
				},
			},
		},
		"aggs": map[string]interface{}{
			"uptime_percentage": map[string]interface{}{
				"weighted_avg": map[string]interface{}{
					"value": map[string]interface{}{
						"field": "status_numeric",
					},
					"weight": map[string]interface{}{
						"field": "interval_since_last_health_check_ms",
					},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return 0, fmt.Errorf("HealthCheckRepo.GetServerUptimePercentage encode query: %w", err)
	}
	res, err := h.es.Search(
		h.es.Search.WithContext(ctx),
		h.es.Search.WithIndex(esHealthCheckIndexName),
		h.es.Search.WithBody(&buf))
	if err != nil {
		return 0, fmt.Errorf("HealthCheckRepo.GetServerUptimePercentage: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e esErrorResponse
		if err = json.NewDecoder(res.Body).Decode(&e); err != nil {
			return 0, fmt.Errorf("HealthCheckRepo.GetServerUptimePercentage decode err response: %w", err)
		}
		return 0, fmt.Errorf("HealthCheckRepo.GetServerUptimePercentage: %w", apperrors.NewElasticSearchError(res.StatusCode, e.Error.Type, e.Error.Reason))
	}

	var uptimeResponse esUptimePercentageResponse
	if err = json.NewDecoder(res.Body).Decode(&uptimeResponse); err != nil {
		return 0, fmt.Errorf("HealthCheckRepo.GetServerUptimePercentage decode response: %w", err)
	}
	return uptimeResponse.Aggregations.UptimePercentage.Value, nil
}

func NewHealthCheckRepository(db *gorm.DB, esClient *elasticsearch.Client) HealthCheckRepository {
	return &healthCheckRepository{
		es: esClient,
	}
}
