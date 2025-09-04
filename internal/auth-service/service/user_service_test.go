package service

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	mockrepository "VCS_SMS_Microservice/internal/auth-service/mocks/repository"
	mockservice "VCS_SMS_Microservice/internal/auth-service/mocks/service"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
	"testing"
)

func hashPassword(password string) string {
	bytes, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes)
}

func TestGetUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUserRepo := mockrepository.NewMockUserRepository(ctrl)
	u := NewUserService(mockUserRepo, nil)
	ctx := context.Background()
	repoErr := errors.New("repo error")
	testCases := []struct {
		name      string
		userEmail string
		sortOrder string
		limit     int
		offset    int
		mock      func()
		output    []model.User
		expectErr bool
	}{
		{
			name:      "Success",
			userEmail: "test@example.com",
			sortOrder: "asc",
			limit:     10,
			offset:    0,
			mock: func() {
				mockUserRepo.EXPECT().
					GetUsers(ctx, "test@example.com", "asc", 10, 0).
					Return([]model.User{{ID: "1", Email: "test@example.com"}}, nil).
					Times(1)
			},
			output:    []model.User{{ID: "1", Email: "test@example.com"}},
			expectErr: false,
		},
		{
			name:      "Repository Error",
			userEmail: "test@example.com",
			sortOrder: "asc",
			limit:     10,
			offset:    0,
			mock: func() {
				mockUserRepo.EXPECT().
					GetUsers(ctx, "test@example.com", "asc", 10, 0).
					Return(nil, repoErr).
					Times(1)
			},
			output:    nil,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			users, err := u.GetUsers(ctx, tc.userEmail, tc.sortOrder, tc.limit, tc.offset)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.output, users)
			}
		})
	}
}

func TestCreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUserRepo := mockrepository.NewMockUserRepository(ctrl)
	mockRoleService := mockservice.NewMockRoleService(ctrl)
	u := NewUserService(mockUserRepo, mockRoleService)
	someError := errors.New("some error")
	ctx := context.Background()
	userToCreate := model.User{
		Email:    "new@example.com",
		Password: "password123",
		Roles:    []model.Role{{ID: "role1"}},
	}
	expectedUser := model.User{
		ID:    "user123",
		Email: "new@example.com",
		Roles: []model.Role{{ID: "role1", Name: "Admin"}}, // Role đã được fetch đầy đủ
	}

	testCases := []struct {
		name        string
		input       model.User
		mock        func()
		output      model.User
		expectedErr bool
	}{
		{
			name:  "Success with roles",
			input: userToCreate,
			mock: func() {
				mockRoleService.EXPECT().GetRoleListByIDs(ctx, []string{"role1"}).
					Return([]model.Role{{ID: "role1", Name: "Admin"}}, nil).
					Times(1)
				mockUserRepo.EXPECT().CreateUser(ctx, gomock.Any()).
					DoAndReturn(func(ctx context.Context, u model.User) (model.User, error) {
						return expectedUser, nil
					}).
					Times(1)
			},
			output:      expectedUser,
			expectedErr: false,
		},
		{
			name: "Success without roles",
			input: model.User{
				Email:    "no-role@example.com",
				Password: "password123",
			},
			mock: func() {
				mockRoleService.EXPECT().GetRoleListByIDs(gomock.Any(), gomock.Any()).Times(0)
				mockUserRepo.EXPECT().CreateUser(ctx, gomock.Any()).
					Return(model.User{ID: "user456", Email: "no-role@example.com"}, nil).
					Times(1)
			},
			output:      model.User{ID: "user456", Email: "no-role@example.com"},
			expectedErr: false,
		},
		{
			name:  "Role service error",
			input: userToCreate,
			mock: func() {
				mockRoleService.EXPECT().GetRoleListByIDs(ctx, []string{"role1"}).
					Return(nil, someError).
					Times(1)
				mockUserRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			output:      model.User{},
			expectedErr: true,
		},
		{
			name:  "Invalid roles error",
			input: userToCreate,
			mock: func() {
				mockRoleService.EXPECT().GetRoleListByIDs(ctx, []string{"role1"}).
					Return([]model.Role{}, nil).
					Times(1)
				mockUserRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			output:      model.User{},
			expectedErr: true,
		},
		{
			name:  "User repo create error",
			input: userToCreate,
			mock: func() {
				mockRoleService.EXPECT().GetRoleListByIDs(ctx, []string{"role1"}).
					Return([]model.Role{{ID: "role1", Name: "Admin"}}, nil).
					Times(1)
				mockUserRepo.EXPECT().CreateUser(ctx, gomock.Any()).
					Return(model.User{}, someError).
					Times(1)
			},
			output:      model.User{},
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()

			createdUser, err := u.CreateUser(ctx, tc.input)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.output.ID, createdUser.ID)
				assert.Equal(t, tc.output.Email, createdUser.Email)
				assert.Equal(t, tc.output.Roles, createdUser.Roles)
			}
		})
	}
}

func TestGetUserByEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mockrepository.NewMockUserRepository(ctrl)
	u := NewUserService(mockUserRepo, nil)
	someError := errors.New("some error")
	ctx := context.Background()
	expectedUser := model.User{ID: "user1", Email: "test@example.com"}

	testCases := []struct {
		name      string
		email     string
		mock      func()
		output    model.User
		expectErr bool
	}{
		{
			name:  "Success",
			email: "test@example.com",
			mock: func() {
				mockUserRepo.EXPECT().GetUserByEmail(ctx, "test@example.com").
					Return(expectedUser, nil).
					Times(1)
			},
			output:    expectedUser,
			expectErr: false,
		},
		{
			name:  "User not found error",
			email: "notfound@example.com",
			mock: func() {
				mockUserRepo.EXPECT().GetUserByEmail(ctx, "notfound@example.com").
					Return(model.User{}, someError).
					Times(1)
			},
			output:    model.User{},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			user, err := u.GetUserByEmail(ctx, tc.email)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.output, user)
			}
		})
	}
}

func TestGetUserById(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mockrepository.NewMockUserRepository(ctrl)
	u := NewUserService(mockUserRepo, nil)
	ctx := context.Background()
	expectedUser := model.User{ID: "user1", Email: "test@example.com"}
	someError := errors.New("some error")
	testCases := []struct {
		name      string
		id        string
		mock      func()
		output    model.User
		expectErr bool
	}{
		{
			name: "Success",
			id:   "user1",
			mock: func() {
				mockUserRepo.EXPECT().GetUserByID(ctx, "user1").
					Return(expectedUser, nil).
					Times(1)
			},
			output:    expectedUser,
			expectErr: false,
		},
		{
			name: "User not found error",
			id:   "notfound",
			mock: func() {
				mockUserRepo.EXPECT().GetUserByID(ctx, "notfound").
					Return(model.User{}, someError).
					Times(1)
			},
			output:    model.User{},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			user, err := u.GetUserById(ctx, tc.id)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.output, user)
			}
		})
	}
}

func TestUpdateUserByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mockrepository.NewMockUserRepository(ctrl)
	mockRoleService := mockservice.NewMockRoleService(ctrl)
	u := NewUserService(mockUserRepo, mockRoleService)
	someError := errors.New("some error")
	ctx := context.Background()
	userToUpdate := model.User{
		ID:    "user1",
		Email: "updated@example.com",
		Roles: []model.Role{{ID: "role1"}},
	}

	testCases := []struct {
		name      string
		input     model.User
		mock      func()
		expectErr bool
	}{
		{
			name:  "Success with roles",
			input: userToUpdate,
			mock: func() {
				mockRoleService.EXPECT().GetRoleListByIDs(ctx, []string{"role1"}).
					Return([]model.Role{{ID: "role1", Name: "Admin"}}, nil).
					Times(1)
				updatedUserWithFullRoles := userToUpdate
				updatedUserWithFullRoles.Roles = []model.Role{{ID: "role1", Name: "Admin"}}
				mockUserRepo.EXPECT().UpdateUserByID(ctx, updatedUserWithFullRoles).
					Return(nil).
					Times(1)
			},
			expectErr: false,
		},
		{
			name: "Success without roles",
			input: model.User{
				ID:    "user1",
				Email: "updated-no-role@example.com",
			},
			mock: func() {
				mockRoleService.EXPECT().GetRoleListByIDs(gomock.Any(), gomock.Any()).Times(0)
				mockUserRepo.EXPECT().UpdateUserByID(ctx, model.User{ID: "user1", Email: "updated-no-role@example.com"}).
					Return(nil).
					Times(1)
			},
			expectErr: false,
		},
		{
			name:  "Role service error",
			input: userToUpdate,
			mock: func() {
				mockRoleService.EXPECT().GetRoleListByIDs(ctx, []string{"role1"}).
					Return(nil, someError).
					Times(1)
				mockUserRepo.EXPECT().UpdateUserByID(gomock.Any(), gomock.Any()).Times(0)
			},
			expectErr: true,
		},
		{
			name:  "Invalid roles error",
			input: userToUpdate,
			mock: func() {
				mockRoleService.EXPECT().GetRoleListByIDs(ctx, []string{"role1"}).
					Return([]model.Role{}, nil). // Trả về slice rỗng
					Times(1)
				mockUserRepo.EXPECT().UpdateUserByID(gomock.Any(), gomock.Any()).Times(0)
			},
			expectErr: true,
		},
		{
			name:  "Repository update error",
			input: userToUpdate,
			mock: func() {
				mockRoleService.EXPECT().GetRoleListByIDs(ctx, []string{"role1"}).
					Return([]model.Role{{ID: "role1", Name: "Admin"}}, nil).
					Times(1)
				mockUserRepo.EXPECT().UpdateUserByID(ctx, gomock.Any()).
					Return(someError).
					Times(1)
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			err := u.UpdateUserByID(ctx, tc.input)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateUserPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mockrepository.NewMockUserRepository(ctrl)
	u := NewUserService(mockUserRepo, nil)
	someError := errors.New("some error")
	ctx := context.Background()
	userID := "user-abc"
	currentPassword := "current-password"
	newPassword := "new-password"
	hashedCurrentPassword := hashPassword(currentPassword)

	testCases := []struct {
		name            string
		currentPassword string
		newPassword     string
		mock            func()
		expectedErr     error
	}{
		{
			name:            "Success",
			currentPassword: currentPassword,
			newPassword:     newPassword,
			mock: func() {
				mockUserRepo.EXPECT().GetUserByID(ctx, userID).
					Return(model.User{ID: userID, Password: hashedCurrentPassword}, nil).
					Times(1)
				mockUserRepo.EXPECT().UpdateUserByID(ctx, gomock.Any()).
					DoAndReturn(func(ctx context.Context, u model.User) error {
						return nil
					}).
					Times(1)
			},
		},
		{
			name:            "Get user by id error",
			currentPassword: currentPassword,
			newPassword:     newPassword,
			mock: func() {
				mockUserRepo.EXPECT().GetUserByID(ctx, userID).
					Return(model.User{}, someError).
					Times(1)
				mockUserRepo.EXPECT().UpdateUserByID(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedErr: someError,
		},
		{
			name:            "Invalid current password",
			currentPassword: "wrong-password",
			newPassword:     newPassword,
			mock: func() {
				mockUserRepo.EXPECT().GetUserByID(ctx, userID).
					Return(model.User{ID: userID, Password: hashedCurrentPassword}, nil).
					Times(1)
				mockUserRepo.EXPECT().UpdateUserByID(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedErr: apperrors.ErrInvalidPassword,
		},
		{
			name:            "Update user by id error",
			currentPassword: currentPassword,
			newPassword:     newPassword,
			mock: func() {
				mockUserRepo.EXPECT().GetUserByID(ctx, userID).
					Return(model.User{ID: userID, Password: hashedCurrentPassword}, nil).
					Times(1)
				mockUserRepo.EXPECT().UpdateUserByID(ctx, gomock.Any()).
					Return(someError).
					Times(1)
			},
			expectedErr: someError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			err := u.UpdateUserPassword(ctx, userID, tc.currentPassword, tc.newPassword)
			if tc.expectedErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
