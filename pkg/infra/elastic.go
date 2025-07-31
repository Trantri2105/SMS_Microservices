package config

import (
	"context"
	"github.com/elastic/go-elasticsearch/v9"
	"time"
)

type ElasticsearchConfig struct {
	Addresses []string
}

func NewElasticSearchConnection(cfg ElasticsearchConfig) (*elasticsearch.Client, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Addresses,
	})

	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = es.Ping(es.Ping.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	return es, nil
}
