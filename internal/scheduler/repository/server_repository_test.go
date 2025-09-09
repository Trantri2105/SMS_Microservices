package repository

import (
	apperrors "VCS_SMS_Microservice/internal/scheduler/errors"
	"VCS_SMS_Microservice/internal/scheduler/model"
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTests(t *testing.T) (ServerRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	assert.NoError(t, err)
	repo := NewServerRepository(gormDB)
	return repo, mock
}

func TestServerRepository_CreateServer(t *testing.T) {
	server := model.Server{
		ID:                  "test-id",
		Ipv4:                "127.0.0.1",
		Port:                8080,
		HealthEndpoint:      "/health",
		HealthCheckInterval: 30,
	}

	testCases := []struct {
		name      string
		mock      func(mock sqlmock.Sqlmock)
		input     model.Server
		expectErr bool
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "servers" ("id","ipv4","port","health_endpoint","health_check_interval","next_health_check_at","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`)).
					WithArgs(sqlmock.AnyArg(), server.Ipv4, server.Port, server.HealthEndpoint, server.HealthCheckInterval, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			input:     server,
			expectErr: false,
		},
		{
			name: "Database Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "servers" ("id","ipv4","port","health_endpoint","health_check_interval","next_health_check_at","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`)).
					WithArgs(sqlmock.AnyArg(), server.Ipv4, server.Port, server.HealthEndpoint, server.HealthCheckInterval, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(errors.New("db error"))
				mock.ExpectRollback()
			},
			input:     server,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := setupTests(t)
			tc.mock(mock)
			ctx := context.Background()

			createdServer, err := repo.CreateServer(ctx, tc.input)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, createdServer.ID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestServerRepository_GetServersForHealthCheck(t *testing.T) {
	serverRows := []string{"id", "ipv4", "port", "health_endpoint", "health_check_interval", "next_health_check_at", "created_at", "updated_at"}
	server1 := model.Server{ID: "server-1", Ipv4: "1.1.1.1", Port: 80, HealthCheckInterval: 30}
	server2 := model.Server{ID: "server-2", Ipv4: "2.2.2.2", Port: 80, HealthCheckInterval: 60}

	testCases := []struct {
		name          string
		mock          func(mock sqlmock.Sqlmock)
		expectedCount int
		expectErr     bool
	}{
		{
			name: "Success Found Servers",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(serverRows).
					AddRow(server1.ID, server1.Ipv4, server1.Port, server1.HealthEndpoint, server1.HealthCheckInterval, time.Now(), time.Now(), time.Now()).
					AddRow(server2.ID, server2.Ipv4, server2.Port, server2.HealthEndpoint, server2.HealthCheckInterval, time.Now(), time.Now(), time.Now())
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers" WHERE next_health_check_at <= NOW()`)).
					WillReturnRows(rows)
			},
			expectedCount: 2,
			expectErr:     false,
		},
		{
			name: "Success No Servers Found",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(serverRows)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers" WHERE next_health_check_at <= NOW()`)).
					WillReturnRows(rows)
			},
			expectedCount: 0,
			expectErr:     false,
		},
		{
			name: "Database Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers" WHERE next_health_check_at <= NOW()`)).
					WillReturnError(errors.New("db error"))
			},
			expectedCount: 0,
			expectErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := setupTests(t)
			tc.mock(mock)
			ctx := context.Background()

			servers, err := repo.GetServersForHealthCheck(ctx)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, servers, tc.expectedCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestServerRepository_UpdateServer(t *testing.T) {
	updatedServerData := model.Server{
		ID:   "server-1",
		Ipv4: "1.2.3.4",
		Port: 8080,
	}
	serverRows := []string{"id", "ipv4", "port"}
	testErr := errors.New("test error")
	testCases := []struct {
		name        string
		mock        func(mock sqlmock.Sqlmock)
		input       model.Server
		ecpectedErr error
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(serverRows).AddRow(updatedServerData.ID, updatedServerData.Ipv4, updatedServerData.Port)
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`UPDATE "servers" SET "id"=$1,"ipv4"=$2,"port"=$3,"updated_at"=$4 WHERE id = $5 RETURNING *`)).
					WithArgs(updatedServerData.ID, updatedServerData.Ipv4, updatedServerData.Port, sqlmock.AnyArg(), updatedServerData.ID).
					WillReturnRows(rows)
				mock.ExpectCommit()
			},
			input: updatedServerData,
		},
		{
			name: "Server Not Found",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`UPDATE "servers" SET "id"=$1,"ipv4"=$2,"port"=$3,"updated_at"=$4 WHERE id = $5 RETURNING *`)).
					WithArgs(updatedServerData.ID, updatedServerData.Ipv4, updatedServerData.Port, sqlmock.AnyArg(), updatedServerData.ID).
					WillReturnRows(sqlmock.NewRows(serverRows))
				mock.ExpectCommit()
			},
			input:       updatedServerData,
			ecpectedErr: apperrors.ErrServerNotFound,
		},
		{
			name: "Database Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`UPDATE "servers" SET "id"=$1,"ipv4"=$2,"port"=$3,"updated_at"=$4 WHERE id = $5 RETURNING *`)).
					WithArgs(updatedServerData.ID, updatedServerData.Ipv4, updatedServerData.Port, sqlmock.AnyArg(), updatedServerData.ID).
					WillReturnError(testErr)
				mock.ExpectRollback()
			},
			input:       updatedServerData,
			ecpectedErr: testErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := setupTests(t)

			tc.mock(mock)
			ctx := context.Background()

			server, err := repo.UpdateServer(ctx, tc.input)

			if tc.ecpectedErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.ecpectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.input.ID, server.ID)
				assert.Equal(t, tc.input.Ipv4, server.Ipv4)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestServerRepository_UpdateServersNextHealthCheckByIds(t *testing.T) {
	serverIds := []string{"server-1", "server-2"}

	testCases := []struct {
		name      string
		mock      func(mock sqlmock.Sqlmock)
		input     []string
		expectErr bool
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "servers" SET "next_health_check_at"=NOW() + (health_check_interval * INTERVAL '1 second'),"updated_at"=$1 WHERE id IN ($2,$3)`)).
					WithArgs(sqlmock.AnyArg(), serverIds[0], serverIds[1]).
					WillReturnResult(sqlmock.NewResult(1, 2))
				mock.ExpectCommit()
			},
			input:     serverIds,
			expectErr: false,
		},
		{
			name: "Database Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "servers" SET "next_health_check_at"=NOW() + (health_check_interval * INTERVAL '1 second'),"updated_at"=$1 WHERE id IN ($2,$3)`)).
					WithArgs(sqlmock.AnyArg(), serverIds[0], serverIds[1]).
					WillReturnError(errors.New("db error"))
				mock.ExpectRollback()
			},
			input:     serverIds,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := setupTests(t)
			tc.mock(mock)
			ctx := context.Background()

			err := repo.UpdateServersNextHealthCheckByIds(ctx, tc.input)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestServerRepository_DeleteServerById(t *testing.T) {
	serverId := "server-to-delete"

	testCases := []struct {
		name      string
		mock      func(mock sqlmock.Sqlmock)
		input     string
		expectErr bool
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "servers" WHERE id = $1`)).
					WithArgs(serverId).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			input:     serverId,
			expectErr: false,
		},
		{
			name: "Database Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "servers" WHERE id = $1`)).
					WithArgs(serverId).
					WillReturnError(errors.New("db error"))
				mock.ExpectRollback()
			},
			input:     serverId,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := setupTests(t)
			tc.mock(mock)
			ctx := context.Background()

			err := repo.DeleteServerById(ctx, tc.input)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
