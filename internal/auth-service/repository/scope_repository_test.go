package repository

import (
	"VCS_SMS_Microservice/internal/auth-service/model"
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"regexp"
	"testing"
)

func newTestScopeRepoWithMockDB(t *testing.T) (ScopeRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	repo := NewScopeRepository(gormDB)
	return repo, mock
}

func TestScopeRepository_GetScopes(t *testing.T) {
	expectedScopes := []model.Scope{
		{ID: "scope-1", Name: "read:users", Description: "Read all users"},
		{ID: "scope-2", Name: "write:users", Description: "Write users"},
	}

	tests := []struct {
		name          string
		scopeName     string
		sortBy        string
		sortOrder     string
		limit         int
		offset        int
		mockSetup     func(mock sqlmock.Sqlmock)
		expectError   bool
		expectedCount int
	}{
		{
			name:      "Success, Get scopes with filter and pagination",
			scopeName: "read:",
			sortBy:    "name",
			sortOrder: "asc",
			limit:     10,
			offset:    0,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "description"}).
					AddRow(expectedScopes[0].ID, expectedScopes[0].Name, expectedScopes[0].Description)

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "scopes" WHERE name LIKE $1 ORDER BY name asc LIMIT $2`)).
					WithArgs("read:%", 10).
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 1,
		},
		{
			name:      "Success, Get scopes without filter",
			scopeName: "",
			sortBy:    "created_at",
			sortOrder: "desc",
			limit:     5,
			offset:    5,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "description"}).
					AddRow(expectedScopes[0].ID, expectedScopes[0].Name, expectedScopes[0].Description).
					AddRow(expectedScopes[1].ID, expectedScopes[1].Name, expectedScopes[1].Description)

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "scopes" ORDER BY created_at desc LIMIT $1 OFFSET $2`)).
					WithArgs(5, 5).
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:      "Error, Database error",
			scopeName: "",
			sortBy:    "name",
			sortOrder: "asc",
			limit:     10,
			offset:    0,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "scopes" ORDER BY name asc LIMIT $1`)).
					WithArgs(10).
					WillReturnError(errors.New("db connection error"))
			},
			expectError:   true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestScopeRepoWithMockDB(t)
			tt.mockSetup(mock)

			scopes, err := repo.GetScopes(context.Background(), tt.scopeName, tt.sortBy, tt.sortOrder, tt.limit, tt.offset)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, scopes)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, scopes)
				assert.Len(t, scopes, tt.expectedCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestScopeRepository_GetScopesListByIDs(t *testing.T) {
	scopeIDs := []string{"scope-1", "scope-2"}
	expectedScopes := []model.Scope{
		{ID: "scope-1", Name: "read:users"},
		{ID: "scope-2", Name: "write:users"},
	}

	tests := []struct {
		name          string
		inputIDs      []string
		mockSetup     func(mock sqlmock.Sqlmock)
		expectError   bool
		expectedCount int
	}{
		{
			name:     "Success - Scopes found",
			inputIDs: scopeIDs,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name"}).
					AddRow(expectedScopes[0].ID, expectedScopes[0].Name).
					AddRow(expectedScopes[1].ID, expectedScopes[1].Name)

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "scopes" WHERE id IN ($1,$2)`)).
					WithArgs(scopeIDs[0], scopeIDs[1]).
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:     "Error - Database error",
			inputIDs: scopeIDs,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "scopes" WHERE id IN ($1,$2)`)).
					WithArgs(scopeIDs[0], scopeIDs[1]).
					WillReturnError(errors.New("db timeout"))
			},
			expectError:   true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestScopeRepoWithMockDB(t)
			tt.mockSetup(mock)

			scopes, err := repo.GetScopesListByIDs(context.Background(), tt.inputIDs)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, scopes)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, scopes)
				assert.Len(t, scopes, tt.expectedCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
