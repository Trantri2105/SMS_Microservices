package infra

import (
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/elasticsearch"
)

func TestNewElasticSearchConnection(t *testing.T) {
	elasticsearchContainer, err := elasticsearch.Run(
		context.Background(),
		"docker.elastic.co/elasticsearch/elasticsearch:8.17.4",
		testcontainers.WithEnv(map[string]string{
			"xpack.security.enabled": "false",
		}),
	)
	if err != nil {
		log.Fatalf("failed to start elasticsearch container: %s", err)
		return
	}
	defer func() {
		if e := testcontainers.TerminateContainer(elasticsearchContainer); e != nil {
			log.Fatalf("failed to terminate container: %s", e)
		}
	}()

	testCases := []struct {
		name      string
		input     ElasticsearchConfig
		expectErr bool
	}{
		{
			name: "valid input",
			input: ElasticsearchConfig{
				Addresses: []string{elasticsearchContainer.Settings.Address},
			},
			expectErr: false,
		},
		{
			name: "invalid input",
			input: ElasticsearchConfig{
				Addresses: []string{"127.0.0.1:8001"},
			},
			expectErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, e := NewElasticSearchConnection(tc.input)
			if tc.expectErr {
				assert.Error(t, e)
			} else {
				assert.NoError(t, e)
				assert.NotNil(t, client)
			}
		})
	}
}
