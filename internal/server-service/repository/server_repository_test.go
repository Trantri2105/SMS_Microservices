package repository

import (
	apperrors "VCS_SMS_Microservice/internal/server-service/errors"
	"VCS_SMS_Microservice/internal/server-service/model"
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock
}

func TestCreateServer(t *testing.T) {
	testErr := errors.New("test error")
	tests := []struct {
		name          string
		input         model.Server
		mockSetup     func(mock sqlmock.Sqlmock, server model.Server)
		expectedError error
	}{
		{
			name: "Success",
			input: model.Server{
				ServerName: "Test Server 1",
				Status:     "active",
				Ipv4:       "127.0.0.1",
			},
			mockSetup: func(mock sqlmock.Sqlmock, server model.Server) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "servers" ("server_name","status","ipv4","port","health_endpoint","health_check_interval","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "id"`)).
					WithArgs(server.ServerName, server.Status, server.Ipv4, server.Port, server.HealthEndpoint, server.HealthCheckInterval, sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("new-uuid-1"))
				mock.ExpectCommit()
			},
			expectedError: nil,
		},
		{
			name: "Error Server Name Already Exists",
			input: model.Server{
				ServerName: "Duplicate Server",
			},
			mockSetup: func(mock sqlmock.Sqlmock, server model.Server) {
				pgErr := &pgconn.PgError{
					Code:           "23505",
					ConstraintName: "servers_server_name_key",
				}
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "servers" ("server_name","status","ipv4","port","health_endpoint","health_check_interval","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "id"`)).
					WillReturnError(pgErr)
				mock.ExpectRollback()
			},
			expectedError: apperrors.ErrServerNameAlreadyExists,
		},
		{
			name: "Error Generic Database Error",
			input: model.Server{
				ServerName: "Error Server",
			},
			mockSetup: func(mock sqlmock.Sqlmock, server model.Server) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "servers" ("server_name","status","ipv4","port","health_endpoint","health_check_interval","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "id"`)).
					WillReturnError(testErr)
				mock.ExpectRollback()
			},
			expectedError: testErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			repo := NewServerRepository(db)
			ctx := context.Background()

			tc.mockSetup(mock, tc.input)

			createdServer, err := repo.CreateServer(ctx, tc.input)

			if tc.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, createdServer.ID)
				assert.Equal(t, tc.input.ServerName, createdServer.ServerName)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetServerById(t *testing.T) {
	serverID := "test-uuid"
	testErr := errors.New("test error")
	expectedServer := model.Server{
		ID:         serverID,
		ServerName: "Found Server",
		Status:     "active",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tests := []struct {
		name          string
		serverID      string
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name:     "Success",
			serverID: serverID,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "server_name", "status", "created_at", "updated_at"}).
					AddRow(expectedServer.ID, expectedServer.ServerName, expectedServer.Status, expectedServer.CreatedAt, expectedServer.UpdatedAt)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers" WHERE id = $1 ORDER BY "servers"."id" LIMIT $2`)).
					WithArgs(serverID, 1).
					WillReturnRows(rows)
			},
			expectedError: nil,
		},
		{
			name:     "Error Not Found",
			serverID: "not-found-uuid",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers" WHERE id = $1 ORDER BY "servers"."id" LIMIT $2`)).
					WithArgs("not-found-uuid", 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedError: apperrors.ErrServerNotFound,
		},
		{
			name:     "Error Generic Database Error",
			serverID: "error-uuid",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers" WHERE id = $1 ORDER BY "servers"."id" LIMIT $2`)).
					WithArgs("error-uuid", 1).
					WillReturnError(testErr)
			},
			expectedError: testErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			repo := NewServerRepository(db)
			ctx := context.Background()

			tc.mockSetup(mock)

			server, err := repo.GetServerById(ctx, tc.serverID)

			if tc.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, expectedServer.ID, server.ID)
				assert.Equal(t, expectedServer.ServerName, server.ServerName)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateServer(t *testing.T) {
	updatedServer := model.Server{
		ID:         "update-uuid",
		ServerName: "Updated Name",
		Status:     "inactive",
	}
	testErr := errors.New("test error")
	tests := []struct {
		name           string
		input          model.Server
		mockSetup      func(mock sqlmock.Sqlmock)
		expectedError  error
		expectedResult model.Server
	}{
		{
			name:  "Success",
			input: updatedServer,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "server_name", "status"}).
					AddRow(updatedServer.ID, updatedServer.ServerName, updatedServer.Status)
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`UPDATE "servers" SET "id"=$1,"server_name"=$2,"status"=$3,"updated_at"=$4 WHERE id = $5 RETURNING *`)).
					WithArgs(updatedServer.ID, updatedServer.ServerName, updatedServer.Status, sqlmock.AnyArg(), updatedServer.ID).
					WillReturnRows(rows)
				mock.ExpectCommit()
			},
			expectedError:  nil,
			expectedResult: updatedServer,
		},
		{
			name:  "Error Not Found",
			input: model.Server{ID: "not-found-uuid", ServerName: "ghost"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`UPDATE "servers" SET "id"=$1,"server_name"=$2,"updated_at"=$3 WHERE id = $4 RETURNING *`)).
					WithArgs("not-found-uuid", "ghost", sqlmock.AnyArg(), "not-found-uuid").
					WillReturnRows(sqlmock.NewRows([]string{}))
				mock.ExpectCommit()
			},
			expectedError: apperrors.ErrServerNotFound,
		},
		{
			name:  "Error Generic Database Error",
			input: updatedServer,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`UPDATE "servers" SET "id"=$1,"server_name"=$2,"status"=$3,"updated_at"=$4 WHERE id = $5 RETURNING *`)).
					WillReturnError(testErr)
				mock.ExpectRollback()
			},
			expectedError: testErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			repo := NewServerRepository(db)
			ctx := context.Background()

			tc.mockSetup(mock)

			result, err := repo.UpdateServer(ctx, tc.input)

			if tc.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedResult.ID, result.ID)
				assert.Equal(t, tc.expectedResult.ServerName, result.ServerName)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetServers(t *testing.T) {
	server1 := model.Server{ID: "uuid-1", ServerName: "Server A", Status: "active"}
	server2 := model.Server{ID: "uuid-2", ServerName: "Server B", Status: "inactive"}

	tests := []struct {
		name       string
		serverName string
		status     string
		mockSetup  func(mock sqlmock.Sqlmock)
		wantCount  int
		wantErr    bool
	}{
		{
			name:       "Success - No filters",
			serverName: "",
			status:     "",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "server_name", "status"}).
					AddRow(server1.ID, server1.ServerName, server1.Status).
					AddRow(server2.ID, server2.ServerName, server2.Status)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers" ORDER BY created_at desc LIMIT $1`)).
					WithArgs(sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:       "Success Filter by server name",
			serverName: "Server A",
			status:     "",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "server_name", "status"}).
					AddRow(server1.ID, server1.ServerName, server1.Status)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers" WHERE server_name LIKE $1 ORDER BY created_at desc LIMIT $2`)).
					WithArgs("Server A%", sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:       "Success Filter by status",
			serverName: "",
			status:     "inactive",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "server_name", "status"}).
					AddRow(server2.ID, server2.ServerName, server2.Status)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers" WHERE status = $1 ORDER BY created_at desc LIMIT $2`)).
					WithArgs("inactive", sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:       "Success Filter by both name and status",
			serverName: "Server A",
			status:     "active",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "server_name", "status"}).
					AddRow(server1.ID, server1.ServerName, server1.Status)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers" WHERE server_name LIKE $1 AND status = $2 ORDER BY created_at desc LIMIT $3`)).
					WithArgs("Server A%", "active", sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:       "Error DB error",
			serverName: "",
			status:     "",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers"`)).
					WillReturnError(errors.New("db find error"))
			},
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			repo := NewServerRepository(db)
			ctx := context.Background()

			tc.mockSetup(mock)

			servers, err := repo.GetServers(ctx, tc.serverName, tc.status, "created_at", "desc", 10, 0)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, servers, tc.wantCount)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestImportServers(t *testing.T) {
	serversToImport := []model.Server{
		{ServerName: "Server1", Status: "active"},
		{ServerName: "Server2", Status: "active"},
		{ServerName: "Server3", Status: "inactive"},
	}

	insertedServersMock := []model.Server{
		{ID: "uuid-1", ServerName: "Server1", Status: "active"},
		{ID: "uuid-2", ServerName: "Server2", Status: "active"},
	}

	tests := []struct {
		name                     string
		input                    []model.Server
		mockSetup                func(mock sqlmock.Sqlmock)
		expectedInsertedCount    int
		expectedNonInsertedCount int
		wantErr                  bool
	}{
		{
			name:  "Success Partial import",
			input: serversToImport,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "server_name", "status"}).
					AddRow(insertedServersMock[0].ID, insertedServersMock[0].ServerName, insertedServersMock[0].Status).
					AddRow(insertedServersMock[1].ID, insertedServersMock[1].ServerName, insertedServersMock[1].Status)

				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "servers" ("server_name","status","ipv4","port","health_endpoint","health_check_interval","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8),($9,$10,$11,$12,$13,$14,$15,$16),($17,$18,$19,$20,$21,$22,$23,$24) ON CONFLICT DO NOTHING RETURNING *`)).
					WillReturnRows(rows)
				mock.ExpectCommit()
			},
			expectedInsertedCount:    2,
			expectedNonInsertedCount: 1,
			wantErr:                  false,
		},
		{
			name:  "Success Empty input",
			input: []model.Server{},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			expectedInsertedCount:    0,
			expectedNonInsertedCount: 0,
			wantErr:                  false,
		},
		{
			name:  "Error Transaction failed",
			input: serversToImport,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "servers"`)).
					WillReturnError(errors.New("transaction error"))
				mock.ExpectRollback()
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			repo := NewServerRepository(db)
			ctx := context.Background()

			tc.mockSetup(mock)

			inserted, nonInserted, err := repo.ImportServers(ctx, tc.input)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, inserted, tc.expectedInsertedCount)
				assert.Len(t, nonInserted, tc.expectedNonInsertedCount)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDeleteServerById(t *testing.T) {
	serverID := "delete-uuid"

	tests := []struct {
		name      string
		serverID  string
		mockSetup func(mock sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name:     "Success",
			serverID: serverID,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "servers" WHERE id = $1`)).
					WithArgs(serverID).
					WillReturnResult(sqlmock.NewResult(0, 1)) // 1 hàng bị ảnh hưởng
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:     "Error DB error",
			serverID: serverID,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "servers" WHERE id = $1`)).
					WithArgs(serverID).
					WillReturnError(errors.New("delete failed"))
				mock.ExpectRollback()
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			repo := NewServerRepository(db)
			ctx := context.Background()

			tc.mockSetup(mock)

			err := repo.DeleteServerById(ctx, tc.serverID)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
