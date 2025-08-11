package infra

import (
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestNewPostgresConnection(t *testing.T) {
	dbName := "test"
	dbUser := "admin"
	dbPassword := "123456"

	postgresContainer, err := postgres.Run(context.Background(),
		"postgres:17.4",
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.WithDatabase(dbName),
		postgres.BasicWaitStrategies(),
	)
	defer func() {
		if e := testcontainers.TerminateContainer(postgresContainer); e != nil {
			log.Fatalf("failed to terminate container: %s", err)
		}
	}()
	if err != nil {
		log.Fatalf("failed to start postgres container: %s", err)
		return
	}

	host, err := postgresContainer.Host(context.Background())
	require.NoError(t, err)

	port, err := postgresContainer.MappedPort(context.Background(), "5432")
	require.NoError(t, err)

	testCases := []struct {
		name        string
		input       PostgresConfig
		expectedErr bool
	}{
		{
			name: "valid input",
			input: PostgresConfig{
				Host:     host,
				Port:     port.Int(),
				User:     dbUser,
				Password: dbPassword,
				DBName:   dbName,
			},
			expectedErr: false,
		},
		{
			name: "invalid input",
			input: PostgresConfig{
				Host:     host,
				Port:     port.Int(),
				User:     dbUser,
				Password: "wrong password",
				DBName:   dbName,
			},
			expectedErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, e := NewPostgresConnection(tc.input)
			if tc.expectedErr {
				assert.Error(t, e)
			} else {
				assert.NoError(t, e)
				assert.NotNil(t, db)
			}
		})
	}
}
