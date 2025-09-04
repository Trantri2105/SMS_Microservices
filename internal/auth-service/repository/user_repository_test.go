package repository

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"regexp"
	"testing"
)

func newTestUserRepoWithMockDB(t *testing.T) (UserRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	repo := NewUserRepository(gormDB)
	return repo, mock
}

func TestUserRepository_CreateUser(t *testing.T) {
	userToCreate := model.User{
		ID:        "new-user-id",
		Email:     "test@example.com",
		Password:  "hashedpassword",
		FirstName: "John",
		LastName:  "Doe",
	}
	dbErr := errors.New("db error")
	tests := []struct {
		name          string
		inputUser     model.User
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name: "Success, User created",
			inputUser: model.User{
				Email:     "test@example.com",
				Password:  "hashedpassword",
				FirstName: "John",
				LastName:  "Doe",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				rows := sqlmock.NewRows([]string{"id"}).AddRow(userToCreate.ID)
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users" ("email","password","first_name","last_name","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "id"`)).
					WithArgs(userToCreate.Email, userToCreate.Password, userToCreate.FirstName, userToCreate.LastName, sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(rows)
				mock.ExpectCommit()
			},
		},
		{
			name: "Error, Email already exists",
			inputUser: model.User{
				Email:     "test@example.com",
				Password:  "hashedpassword",
				FirstName: "John",
				LastName:  "Doe",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				pgErr := &pgconn.PgError{Code: "23505", ConstraintName: "users_email_key"}
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users" ("email","password","first_name","last_name","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "id"`)).
					WillReturnError(pgErr)
				mock.ExpectRollback()
			},
			expectedError: apperrors.ErrUserMailAlreadyExists,
		},
		{
			name: "Error, Generic database error",
			inputUser: model.User{
				Email:     "test@example.com",
				Password:  "hashedpassword",
				FirstName: "John",
				LastName:  "Doe",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users" ("email","password","first_name","last_name","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "id"`)).
					WillReturnError(dbErr)
				mock.ExpectRollback()
			},
			expectedError: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestUserRepoWithMockDB(t)
			tt.mockSetup(mock)

			createdUser, err := repo.CreateUser(context.Background(), tt.inputUser)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, userToCreate.ID, createdUser.ID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetUserByEmail(t *testing.T) {
	userEmail := "found@example.com"
	userID := "user-123"
	roleID := "role-456"
	scopeID := "scope-123"
	dbErr := errors.New("db error")
	tests := []struct {
		name          string
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name: "Success, User found with roles and scopes",
			mockSetup: func(mock sqlmock.Sqlmock) {
				userRows := sqlmock.NewRows([]string{"id", "email"}).AddRow(userID, userEmail)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1 ORDER BY "users"."id" LIMIT $2`)).
					WithArgs(userEmail, 1).WillReturnRows(userRows)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user_roles" WHERE "user_roles"."user_id" = $1`)).
					WithArgs(userID).
					WillReturnRows(sqlmock.NewRows([]string{"user_id", "role_id"}).AddRow(userID, roleID))
				roleRows := sqlmock.NewRows([]string{"id", "name"}).AddRow(roleID, "Admin")
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE "roles"."id" = $1`)).
					WithArgs(roleID).WillReturnRows(roleRows)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "role_scopes" WHERE "role_scopes"."role_id" = $1`)).
					WithArgs(roleID).
					WillReturnRows(sqlmock.NewRows([]string{"role_id", "scope_id"}).AddRow(roleID, scopeID))
				scopeRows := sqlmock.NewRows([]string{"id", "name"}).AddRow(scopeID, "read:all")
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "scopes" WHERE "scopes"."id" = $1`)).
					WithArgs(scopeID).WillReturnRows(scopeRows)
			},
		},
		{
			name: "Error, User not found",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1 ORDER BY "users"."id" LIMIT $2`)).
					WithArgs(userEmail, 1).WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedError: apperrors.ErrUserNotFound,
		},
		{
			name: "Error, Database error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1 ORDER BY "users"."id" LIMIT $2`)).
					WithArgs(userEmail, 1).WillReturnError(dbErr)
			},
			expectedError: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestUserRepoWithMockDB(t)
			tt.mockSetup(mock)

			_, err := repo.GetUserByEmail(context.Background(), userEmail)

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

func TestUserRepository_UpdateUserByID(t *testing.T) {
	userToUpdate := model.User{
		ID:        "user-to-update",
		Email:     "update@example.com",
		FirstName: "Jane",
		Roles:     []model.Role{{ID: "new-role-1"}},
	}

	tests := []struct {
		name          string
		inputUser     model.User
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name:      "Success, Update user info and roles",
			inputUser: userToUpdate,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "email"=$1,"first_name"=$2,"updated_at"=$3 WHERE "id" = $4`)).
					WithArgs(userToUpdate.Email, userToUpdate.FirstName, sqlmock.AnyArg(), userToUpdate.ID).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "updated_at"=$1 WHERE "id" = $2`)).
					WithArgs(sqlmock.AnyArg(), userToUpdate.ID).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "user_roles" ("user_id","role_id") VALUES ($1,$2) ON CONFLICT DO NOTHING RETURNING "user_id","role_id"`)).
					WithArgs(userToUpdate.ID, userToUpdate.Roles[0].ID).
					WillReturnRows(sqlmock.NewRows([]string{"role_id", "scope_id"}).AddRow(userToUpdate.ID, userToUpdate.Roles[0].ID))
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "user_roles" WHERE "user_roles"."user_id" = $1 AND "user_roles"."role_id" <> $2`)).
					WithArgs(userToUpdate.ID, userToUpdate.Roles[0].ID).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name:      "Success, Update user info without roles",
			inputUser: model.User{ID: "user-to-update", Email: "update@example.com"}, // Roles slice is empty
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "email"=$1,"updated_at"=$2 WHERE "id" = $3`)).
					WithArgs("update@example.com", sqlmock.AnyArg(), "user-to-update").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name:      "Error, User not found",
			inputUser: userToUpdate,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "email"=$1,"first_name"=$2,"updated_at"=$3 WHERE "id" = $4`)).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectRollback()
			},
			expectedError: apperrors.ErrUserNotFound,
		},
		{
			name:      "Error - Email already exists",
			inputUser: userToUpdate,
			mockSetup: func(mock sqlmock.Sqlmock) {
				pgErr := &pgconn.PgError{Code: "23505", ConstraintName: "users_email_key"}
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "email"=$1,"first_name"=$2,"updated_at"=$3 WHERE "id" = $4`)).
					WillReturnError(pgErr)
				mock.ExpectRollback()
			},
			expectedError: apperrors.ErrUserMailAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestUserRepoWithMockDB(t)
			tt.mockSetup(mock)

			err := repo.UpdateUserByID(context.Background(), tt.inputUser)

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

func TestUserRepository_GetUserByID(t *testing.T) {
	userID := "user-123"
	roleID := "role-456"
	userEmail := "user@example.com"
	scopeID := "scope-123"
	dbErr := errors.New("db error")
	tests := []struct {
		name          string
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name: "Success, User found with roles and scopes",
			mockSetup: func(mock sqlmock.Sqlmock) {
				userRows := sqlmock.NewRows([]string{"id", "email"}).AddRow(userID, userEmail)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
					WithArgs(userID, 1).WillReturnRows(userRows)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user_roles" WHERE "user_roles"."user_id" = $1`)).
					WithArgs(userID).
					WillReturnRows(sqlmock.NewRows([]string{"user_id", "role_id"}).AddRow(userID, roleID))
				roleRows := sqlmock.NewRows([]string{"id", "name"}).AddRow(roleID, "Admin")
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE "roles"."id" = $1`)).
					WithArgs(roleID).WillReturnRows(roleRows)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "role_scopes" WHERE "role_scopes"."role_id" = $1`)).
					WithArgs(roleID).
					WillReturnRows(sqlmock.NewRows([]string{"role_id", "scope_id"}).AddRow(roleID, scopeID))
				scopeRows := sqlmock.NewRows([]string{"id", "name"}).AddRow(scopeID, "read:all")
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "scopes" WHERE "scopes"."id" = $1`)).
					WithArgs(scopeID).WillReturnRows(scopeRows)
			},
		},
		{
			name: "Error, User not found",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
					WithArgs(userID, 1).WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedError: apperrors.ErrUserNotFound,
		},
		{
			name: "Error, Database error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
					WithArgs(userID, 1).WillReturnError(dbErr)
			},
			expectedError: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestUserRepoWithMockDB(t)
			tt.mockSetup(mock)

			_, err := repo.GetUserByID(context.Background(), userID)

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

func TestUserRepository_GetUsers(t *testing.T) {
	tests := []struct {
		name          string
		userEmail     string
		sortOrder     string
		limit         int
		offset        int
		mockSetup     func(mock sqlmock.Sqlmock)
		expectError   bool
		expectedCount int
	}{
		{
			name:      "Success - Get users with email filter",
			userEmail: "test@",
			sortOrder: "asc",
			limit:     10,
			offset:    0,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email"}).
					AddRow("user-1", "test@a.com").
					AddRow("user-2", "test@b.com")
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email LIKE $1 ORDER BY created_at asc LIMIT $2`)).
					WithArgs("test@%", 10).
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:      "Success - Get users without email filter",
			userEmail: "",
			sortOrder: "desc",
			limit:     5,
			offset:    5,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email"}).
					AddRow("user-3", "another@c.com")
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" ORDER BY created_at desc LIMIT $1 OFFSET $2`)).
					WithArgs(5, 5).
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 1,
		},
		{
			name:      "Success - No users found",
			userEmail: "nobody@",
			sortOrder: "asc",
			limit:     10,
			offset:    0,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email"})
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email LIKE $1 ORDER BY created_at asc LIMIT $2`)).
					WithArgs("nobody@%", 10).
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 0,
		},
		{
			name:      "Error - Database error",
			userEmail: "",
			sortOrder: "asc",
			limit:     10,
			offset:    0,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" ORDER BY created_at asc LIMIT $1`)).
					WithArgs(10).
					WillReturnError(errors.New("db query failed"))
			},
			expectError:   true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newTestUserRepoWithMockDB(t)
			tt.mockSetup(mock)

			users, err := repo.GetUsers(context.Background(), tt.userEmail, tt.sortOrder, tt.limit, tt.offset)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, users, tt.expectedCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
