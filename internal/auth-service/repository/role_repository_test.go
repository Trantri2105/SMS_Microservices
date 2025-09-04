package repository

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"regexp"
	"testing"
)

func newTestRepoWithMockDB(t *testing.T) (RoleRepository, sqlmock.Sqlmock) {
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

	repo := NewRoleRepository(gormDB)
	return repo, mock
}

func TestRoleRepository_GetRoleByID(t *testing.T) {
	roleID := "role-123"
	expectedRole := model.Role{
		ID:          roleID,
		Name:        "Admin",
		Description: "Administrator role",
		Scopes: []model.Scope{
			{ID: "scope-1", Name: "read:users"},
		},
	}
	dbErr := errors.New("db error")

	tests := []struct {
		name          string
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedRole  model.Role
		expectedError error
	}{
		{
			name: "Success, Role found with scopes",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "description"}).
					AddRow(expectedRole.ID, expectedRole.Name, expectedRole.Description)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE id = $1 ORDER BY "roles"."id" LIMIT $2`)).
					WithArgs(roleID, 1).
					WillReturnRows(rows)
				roleScopeRows := sqlmock.NewRows([]string{"role_id", "scope_id"}).AddRow("role-123", "scope-1")
				scopeRows := sqlmock.NewRows([]string{"id", "name"}).
					AddRow("scope-1", "read:users")
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "role_scopes" WHERE "role_scopes"."role_id" = $1`)).WithArgs(roleID).
					WillReturnRows(roleScopeRows)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "scopes" WHERE "scopes"."id" = $1`)).
					WithArgs("scope-1").
					WillReturnRows(scopeRows)
			},
			expectedRole:  expectedRole,
			expectedError: nil,
		},
		{
			name: "Error, Role not found",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE id = $1 ORDER BY "roles"."id" LIMIT $2`)).
					WithArgs(roleID, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedRole:  model.Role{},
			expectedError: apperrors.ErrRoleNotFound,
		},
		{
			name: "Error, Database error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE id = $1 ORDER BY "roles"."id" LIMIT $2`)).
					WithArgs(roleID, 1).
					WillReturnError(dbErr)
			},
			expectedRole:  model.Role{},
			expectedError: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestRepoWithMockDB(t)
			tt.mockSetup(mock)

			role, err := repo.GetRoleByID(context.Background(), roleID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRole.ID, role.ID)
				assert.Equal(t, tt.expectedRole.Name, role.Name)
				assert.Equal(t, len(tt.expectedRole.Scopes), len(role.Scopes))
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRoleRepository_CreateRole(t *testing.T) {
	roleToCreate := model.Role{
		ID:          "new-role-id",
		Name:        "Editor",
		Description: "Content Editor",
	}
	dbErr := errors.New("db error")
	tests := []struct {
		name          string
		inputRole     model.Role
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name:      "Success, Role created",
			inputRole: model.Role{Name: "Editor", Description: "Content Editor"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				rows := sqlmock.NewRows([]string{"id"}).AddRow(roleToCreate.ID)
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "roles" ("name","description","created_at","updated_at") VALUES ($1,$2,$3,$4) RETURNING "id"`)).
					WithArgs(roleToCreate.Name, roleToCreate.Description, sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(rows)
				mock.ExpectCommit()
			},
		},
		{
			name:      "Error, Role name already exists",
			inputRole: model.Role{Name: "Editor", Description: "Content Editor"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				pgErr := &pgconn.PgError{
					Code:           pgerrcode.UniqueViolation,
					ConstraintName: "roles_name_key",
				}
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "roles" ("name","description","created_at","updated_at") VALUES ($1,$2,$3,$4) RETURNING "id"`)).
					WithArgs(roleToCreate.Name, roleToCreate.Description, sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(pgErr)
				mock.ExpectRollback()
			},
			expectedError: apperrors.ErrRoleNameAlreadyExists,
		},
		{
			name:      "Error, Generic database error",
			inputRole: model.Role{Name: "Editor", Description: "Content Editor"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "roles" ("name","description","created_at","updated_at") VALUES ($1,$2,$3,$4) RETURNING "id"`)).
					WithArgs(roleToCreate.Name, roleToCreate.Description, sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(dbErr)
				mock.ExpectRollback()
			},
			expectedError: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestRepoWithMockDB(t)
			tt.mockSetup(mock)

			createdRole, err := repo.CreateRole(context.Background(), tt.inputRole)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, roleToCreate.ID, createdRole.ID)
				assert.Equal(t, roleToCreate.Name, createdRole.Name)
				assert.Equal(t, roleToCreate.Description, createdRole.Description)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRoleRepository_UpdateRoleByID(t *testing.T) {
	roleToUpdate := model.Role{
		ID:          "role-to-update",
		Name:        "Updated Name",
		Description: "Updated Desc",
		Scopes: []model.Scope{
			{ID: "scope-new", Name: "write:posts"},
		},
	}
	tests := []struct {
		name          string
		inputRole     model.Role
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name:      "Success, Update role and scopes",
			inputRole: roleToUpdate,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "roles" SET "name"=$1,"description"=$2,"updated_at"=$3 WHERE "id" = $4`)).
					WithArgs(roleToUpdate.Name, roleToUpdate.Description, sqlmock.AnyArg(), roleToUpdate.ID).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "roles" SET "updated_at"=$1 WHERE "id" = $2`)).
					WithArgs(sqlmock.AnyArg(), roleToUpdate.ID).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "role_scopes" ("role_id","scope_id") VALUES ($1,$2) ON CONFLICT DO NOTHING RETURNING "role_id","scope_id"`)).
					WithArgs(roleToUpdate.ID, "scope-new").
					WillReturnRows(sqlmock.NewRows([]string{"role_id", "scope_id"}).AddRow(roleToUpdate.ID, "scope-new"))
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "role_scopes" WHERE "role_scopes"."role_id" = $1 AND "role_scopes"."scope_id" <> $2`)).
					WithArgs(roleToUpdate.ID, "scope-new").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name: "Success, Update role without scopes",
			inputRole: model.Role{
				ID:   "role-to-update",
				Name: "Updated Name",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "roles" SET "name"=$1,"updated_at"=$2 WHERE "id" = $3`)).
					WithArgs(roleToUpdate.Name, sqlmock.AnyArg(), roleToUpdate.ID).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name:      "Error, Role not found",
			inputRole: roleToUpdate,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "roles" SET "name"=$1,"description"=$2,"updated_at"=$3 WHERE "id" = $4`)).
					WithArgs(roleToUpdate.Name, roleToUpdate.Description, sqlmock.AnyArg(), roleToUpdate.ID).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectRollback()
			},
			expectedError: apperrors.ErrRoleNotFound,
		},
		{
			name:      "Error, Role name already exists",
			inputRole: roleToUpdate,
			mockSetup: func(mock sqlmock.Sqlmock) {
				pgErr := &pgconn.PgError{
					Code:           pgerrcode.UniqueViolation,
					ConstraintName: "roles_name_key",
				}
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "roles" SET "name"=$1,"description"=$2,"updated_at"=$3 WHERE "id" = $4`)).
					WithArgs(roleToUpdate.Name, roleToUpdate.Description, sqlmock.AnyArg(), roleToUpdate.ID).
					WillReturnError(pgErr)
				mock.ExpectRollback()
			},
			expectedError: apperrors.ErrRoleNameAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestRepoWithMockDB(t)
			tt.mockSetup(mock)

			err := repo.UpdateRoleByID(context.Background(), tt.inputRole)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRoleRepository_DeleteRoleByID(t *testing.T) {
	roleID := "role-to-delete"

	tests := []struct {
		name        string
		mockSetup   func(mock sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name: "Success, Role deleted",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "roles" WHERE id = $1`)).
					WithArgs(roleID).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name: "Error, Database error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "roles" WHERE id = $1`)).
					WithArgs(roleID).
					WillReturnError(errors.New("db error"))
				mock.ExpectRollback()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestRepoWithMockDB(t)
			tt.mockSetup(mock)

			err := repo.DeleteRoleByID(context.Background(), roleID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRoleRepository_GetRoles(t *testing.T) {
	tests := []struct {
		name        string
		roleName    string
		sortBy      string
		sortOrder   string
		limit       int
		offset      int
		mockSetup   func(mock sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name:      "Success, Get roles with filter and pagination",
			roleName:  "A",
			sortBy:    "name",
			sortOrder: "asc",
			limit:     10,
			offset:    0,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name"}).
					AddRow("role-1", "Admin").
					AddRow("role-2", "Accountant")
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE name LIKE $1 ORDER BY name asc LIMIT $2`)).
					WithArgs("A%", 10).
					WillReturnRows(rows)
			},
			expectError: false,
		},
		{
			name:      "Success, Get roles without filter",
			roleName:  "",
			sortBy:    "created_at",
			sortOrder: "desc",
			limit:     5,
			offset:    5,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name"})
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" ORDER BY created_at desc LIMIT $1 OFFSET $2`)).
					WithArgs(5, 5).
					WillReturnRows(rows)
			},
			expectError: false,
		},
		{
			name:      "Error - Database error",
			roleName:  "",
			sortBy:    "name",
			sortOrder: "asc",
			limit:     10,
			offset:    0,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" ORDER BY name asc LIMIT $1`)).
					WithArgs(10).
					WillReturnError(errors.New("db error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestRepoWithMockDB(t)
			tt.mockSetup(mock)

			roles, err := repo.GetRoles(context.Background(), tt.roleName, tt.sortBy, tt.sortOrder, tt.limit, tt.offset)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, roles)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, roles)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRoleRepository_GetRolesListByIDs(t *testing.T) {
	roleIDs := []string{"role-1", "role-2"}
	expectedRoles := []model.Role{
		{ID: "role-1", Name: "Admin"},
		{ID: "role-2", Name: "Editor"},
	}

	tests := []struct {
		name          string
		inputIDs      []string
		mockSetup     func(mock sqlmock.Sqlmock)
		expectError   bool
		expectedCount int
	}{
		{
			name:     "Success - Roles found",
			inputIDs: roleIDs,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name"}).
					AddRow(expectedRoles[0].ID, expectedRoles[0].Name).
					AddRow(expectedRoles[1].ID, expectedRoles[1].Name)

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE id IN ($1,$2)`)).
					WithArgs(roleIDs[0], roleIDs[1]).
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:     "Success - No roles found for given IDs",
			inputIDs: []string{"role-998", "role-999"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name"})
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE id IN ($1,$2)`)).
					WithArgs("role-998", "role-999").
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 0,
		},
		{
			name:     "Error - Database error",
			inputIDs: roleIDs,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE id IN ($1,$2)`)).
					WithArgs(roleIDs[0], roleIDs[1]).
					WillReturnError(errors.New("db error"))
			},
			expectError:   true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestRepoWithMockDB(t)
			tt.mockSetup(mock)

			roles, err := repo.GetRolesListByIDs(context.Background(), tt.inputIDs)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, roles, tt.expectedCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
