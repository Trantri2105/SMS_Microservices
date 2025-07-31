package config

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"log"
	"testing"
)

func TestNewRedisConnection(t *testing.T) {
	redisContainer, err := tcredis.Run(context.Background(), "redis:latest")
	if err != nil {
		log.Fatalf("error creating redis: %s", err)
	}
	defer func() {
		if err := testcontainers.TerminateContainer(redisContainer); err != nil {
			log.Fatalf("failed to terminate container: %s", err)
		}
	}()

	host, err := redisContainer.Host(context.Background())
	require.NoError(t, err)

	port, err := redisContainer.MappedPort(context.Background(), "6379")
	require.NoError(t, err)

	testCases := []struct {
		name        string
		input       RedisConfig
		expectedErr bool
	}{
		{
			name: "valid config",
			input: RedisConfig{
				Host: host,
				Port: port.Int(),
			},
			expectedErr: false,
		},
		{
			name: "invalid config",
			input: RedisConfig{
				Host: host,
				Port: 8001,
			},
			expectedErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, e := NewRedisConnection(tc.input)
			if tc.expectedErr {
				assert.Error(t, e)
			} else {
				assert.NoError(t, e)
				assert.NotNil(t, client)
			}
		})
	}
}
