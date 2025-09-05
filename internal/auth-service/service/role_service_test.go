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
	"testing"
)

func TestNewRoleService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoleRepo := mockrepository.NewMockRoleRepository(ctrl)
	mockScopeService := mockservice.NewMockScopeService(ctrl)

	service := NewRoleService(mockRoleRepo, mockScopeService)
	assert.NotNil(t, service)
}

func TestRoleService_CreateRole(t *testing.T) {
	ctx := context.Background()
	testError := errors.New("test repository error")

	testScopes := []model.Scope{{ID: "scope1"}, {ID: "scope2"}}
	testRoleWithScopes := model.Role{Name: "Admin", Scopes: testScopes}
	testRoleWithoutScopes := model.Role{Name: "Guest"}

	tests := []struct {
		name        string
		role        model.Role
		setupMocks  func(mockRoleRepo *mockrepository.MockRoleRepository, mockScopeService *mockservice.MockScopeService)
		output      model.Role
		expectedErr error
	}{
		{
			name: "Success Create role with valid scopes",
			role: testRoleWithScopes,
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository, mockScopeService *mockservice.MockScopeService) {
				scopeIDs := []string{"scope1", "scope2"}
				mockScopeService.EXPECT().GetScopesByIDs(gomock.Any(), gomock.InAnyOrder(scopeIDs)).Return(testScopes, nil)
				mockRoleRepo.EXPECT().CreateRole(gomock.Any(), testRoleWithScopes).Return(testRoleWithScopes, nil)
			},
			output:      testRoleWithScopes,
			expectedErr: nil,
		},
		{
			name: "Success Create role without scopes",
			role: testRoleWithoutScopes,
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository, mockScopeService *mockservice.MockScopeService) {
				mockRoleRepo.EXPECT().CreateRole(gomock.Any(), testRoleWithoutScopes).Return(testRoleWithoutScopes, nil)
			},
			output:      testRoleWithoutScopes,
			expectedErr: nil,
		},
		{
			name: "Failure Invalid scopes (mismatched length)",
			role: testRoleWithScopes,
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository, mockScopeService *mockservice.MockScopeService) {
				scopeIDs := []string{"scope1", "scope2"}
				mockScopeService.EXPECT().GetScopesByIDs(gomock.Any(), gomock.InAnyOrder(scopeIDs)).Return([]model.Scope{{ID: "scope1"}}, nil)
			},
			output:      model.Role{},
			expectedErr: apperrors.ErrInvalidScopes,
		},
		{
			name: "Failure Scope service returns error",
			role: testRoleWithScopes,
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository, mockScopeService *mockservice.MockScopeService) {
				scopeIDs := []string{"scope1", "scope2"}
				mockScopeService.EXPECT().GetScopesByIDs(gomock.Any(), gomock.InAnyOrder(scopeIDs)).Return(nil, testError)
			},
			output:      model.Role{},
			expectedErr: testError,
		},
		{
			name: "Failure Role repository CreateRole returns error",
			role: testRoleWithoutScopes,
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository, mockScopeService *mockservice.MockScopeService) {
				mockRoleRepo.EXPECT().CreateRole(gomock.Any(), testRoleWithoutScopes).Return(model.Role{}, testError)
			},
			output:      model.Role{},
			expectedErr: testError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRoleRepo := mockrepository.NewMockRoleRepository(ctrl)
			mockScopeService := mockservice.NewMockScopeService(ctrl)
			tt.setupMocks(mockRoleRepo, mockScopeService)
			s := NewRoleService(mockRoleRepo, mockScopeService)
			got, err := s.CreateRole(ctx, tt.role)
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.output, got)
		})
	}
}

func TestRoleService_UpdateRoleByID(t *testing.T) {
	ctx := context.Background()
	testError := errors.New("test repository error")

	sampleScopes := []model.Scope{{ID: "scope1"}}
	sampleRole := model.Role{ID: "role1", Name: "Updated Role", Scopes: sampleScopes}

	tests := []struct {
		name        string
		role        model.Role
		setupMocks  func(mockRoleRepo *mockrepository.MockRoleRepository, mockScopeService *mockservice.MockScopeService)
		expectedErr error
	}{
		{
			name: "Success Update role with valid scopes",
			role: sampleRole,
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository, mockScopeService *mockservice.MockScopeService) {
				mockScopeService.EXPECT().GetScopesByIDs(gomock.Any(), []string{"scope1"}).Return(sampleScopes, nil)
				mockRoleRepo.EXPECT().UpdateRoleByID(gomock.Any(), sampleRole).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name: "Success Update role without scopes",
			role: model.Role{ID: "role1", Name: "No Scopes Role"},
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository, mockScopeService *mockservice.MockScopeService) {
				mockRoleRepo.EXPECT().UpdateRoleByID(gomock.Any(), model.Role{ID: "role1", Name: "No Scopes Role"}).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name: "Failure Invalid scopes",
			role: sampleRole,
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository, mockScopeService *mockservice.MockScopeService) {
				mockScopeService.EXPECT().GetScopesByIDs(gomock.Any(), []string{"scope1"}).Return(nil, apperrors.ErrInvalidScopes)
			},
			expectedErr: apperrors.ErrInvalidScopes,
		},
		{
			name: "Failure - Repository returns error",
			role: sampleRole,
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository, mockScopeService *mockservice.MockScopeService) {
				mockScopeService.EXPECT().GetScopesByIDs(gomock.Any(), []string{"scope1"}).Return(sampleScopes, nil)
				mockRoleRepo.EXPECT().UpdateRoleByID(gomock.Any(), sampleRole).Return(testError)
			},
			expectedErr: testError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRoleRepo := mockrepository.NewMockRoleRepository(ctrl)
			mockScopeService := mockservice.NewMockScopeService(ctrl)

			tt.setupMocks(mockRoleRepo, mockScopeService)
			s := NewRoleService(mockRoleRepo, mockScopeService)

			err := s.UpdateRoleByID(ctx, tt.role)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRoleService_DeleteRoleByID(t *testing.T) {
	ctx := context.Background()
	roleID := "test-role-id"
	dbError := errors.New("db error")

	tests := []struct {
		name        string
		inputID     string
		setupMocks  func(mockRoleRepo *mockrepository.MockRoleRepository)
		expectedErr error
	}{
		{
			name:    "Success",
			inputID: roleID,
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository) {
				mockRoleRepo.EXPECT().DeleteRoleByID(ctx, roleID).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name:    "Failure",
			inputID: roleID,
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository) {
				mockRoleRepo.EXPECT().DeleteRoleByID(ctx, roleID).Return(dbError)
			},
			expectedErr: dbError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRoleRepo := mockrepository.NewMockRoleRepository(ctrl)
			mockScopeService := mockservice.NewMockScopeService(ctrl)
			tt.setupMocks(mockRoleRepo)
			s := NewRoleService(mockRoleRepo, mockScopeService)
			err := s.DeleteRoleByID(ctx, tt.inputID)
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRoleService_GetRoles(t *testing.T) {
	ctx := context.Background()
	testError := errors.New("database error")
	expectedRoles := []model.Role{{ID: "1", Name: "Admin"}, {ID: "2", Name: "User"}}

	type args struct {
		roleName  string
		sortBy    string
		sortOrder string
		limit     int
		offset    int
	}

	tests := []struct {
		name       string
		args       args
		setupMocks func(mockRoleRepo *mockrepository.MockRoleRepository)
		output     []model.Role
		expectErr  bool
	}{
		{
			name: "Success Get roles with default pagination",
			args: args{limit: 10, offset: 0},
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository) {
				mockRoleRepo.EXPECT().GetRoles(ctx, "", "", "", 10, 0).Return(expectedRoles, nil)
			},
			output:    expectedRoles,
			expectErr: false,
		},
		{
			name: "Failure Repository returns an error",
			args: args{limit: 10, offset: 0},
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository) {
				mockRoleRepo.EXPECT().GetRoles(ctx, "", "", "", 10, 0).Return(nil, testError)
			},
			output:    nil,
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockRoleRepo := mockrepository.NewMockRoleRepository(ctrl)
			s := NewRoleService(mockRoleRepo, nil)
			tt.setupMocks(mockRoleRepo)
			got, err := s.GetRoles(ctx, tt.args.roleName, tt.args.sortBy, tt.args.sortOrder, tt.args.limit, tt.args.offset)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.output, got)
		})
	}
}

func TestRoleService_GetRoleByID(t *testing.T) {
	ctx := context.Background()
	testError := errors.New("database error")
	roleID := "test-role-id"
	expectedRole := model.Role{ID: roleID, Name: "Admin"}
	tests := []struct {
		name        string
		setupMocks  func(mockRoleRepo *mockrepository.MockRoleRepository)
		output      model.Role
		expectedErr error
	}{
		{
			name: "Success Found role by ID",
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository) {
				mockRoleRepo.EXPECT().GetRoleByID(ctx, roleID).Return(expectedRole, nil)
			},
			output:      expectedRole,
			expectedErr: nil,
		},
		{
			name: "Failure Repository returns an error",
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository) {
				mockRoleRepo.EXPECT().GetRoleByID(ctx, roleID).Return(model.Role{}, testError)
			},
			output:      model.Role{},
			expectedErr: testError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoleRepo := mockrepository.NewMockRoleRepository(ctrl)
			s := NewRoleService(mockRoleRepo, nil)
			tt.setupMocks(mockRoleRepo)
			got, err := s.GetRoleByID(ctx, roleID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expectedRole, got)
			}
		})
	}
}

func TestRoleService_GetRoleListByIDs(t *testing.T) {
	ctx := context.Background()
	testError := errors.New("database error")
	roleIDs := []string{"id1", "id2"}
	expectedRoles := []model.Role{{ID: "id1"}, {ID: "id2"}}

	tests := []struct {
		name       string
		setupMocks func(mockRoleRepo *mockrepository.MockRoleRepository)
		output     []model.Role
		expectErr  bool
	}{
		{
			name: "Success Found roles by list of IDs",
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository) {
				mockRoleRepo.EXPECT().GetRolesListByIDs(ctx, roleIDs).Return(expectedRoles, nil)
			},
			output:    expectedRoles,
			expectErr: false,
		},
		{
			name: "Failure Repository returns an error",
			setupMocks: func(mockRoleRepo *mockrepository.MockRoleRepository) {
				mockRoleRepo.EXPECT().GetRolesListByIDs(ctx, roleIDs).Return(nil, testError)
			},
			output:    nil,
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoleRepo := mockrepository.NewMockRoleRepository(ctrl)
			s := NewRoleService(mockRoleRepo, nil)

			tt.setupMocks(mockRoleRepo)

			got, err := s.GetRoleListByIDs(ctx, roleIDs)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.output, got)
		})
	}
}
