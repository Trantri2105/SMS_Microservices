package service

import (
	mockrepository "VCS_SMS_Microservice/internal/auth-service/mocks/repository"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestScopeService_GetScopesList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mockrepository.NewMockScopeRepository(ctrl)

	s := NewScopeService(mockRepo)

	ctx := context.Background()
	expectedScopes := []model.Scope{{ID: "scope-1", Name: "read:users"}}
	repoError := errors.New("database connection failed")

	testCases := []struct {
		name           string
		scopeName      string
		sortBy         string
		sortOrder      string
		limit          int
		offset         int
		setupMock      func()
		expectedResult []model.Scope
		expectedError  error
	}{
		{
			name:      "Success, Should return scopes from repository",
			scopeName: "read",
			sortBy:    "name",
			sortOrder: "asc",
			limit:     10,
			offset:    0,
			setupMock: func() {
				mockRepo.EXPECT().
					GetScopes(ctx, "read", "name", "asc", 10, 0).
					Return(expectedScopes, nil)
			},
			expectedResult: expectedScopes,
			expectedError:  nil,
		},
		{
			name:      "Failure, Should return error when repository fails",
			scopeName: "read",
			sortBy:    "name",
			sortOrder: "asc",
			limit:     10,
			offset:    0,
			setupMock: func() {
				mockRepo.EXPECT().
					GetScopes(ctx, "read", "name", "asc", 10, 0).
					Return(nil, repoError)
			},
			expectedResult: nil,
			expectedError:  repoError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMock()
			scopes, err := s.GetScopesList(ctx, tc.scopeName, tc.sortBy, tc.sortOrder, tc.limit, tc.offset)
			assert.Equal(t, tc.expectedResult, scopes)
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestScopeService_GetScopesByIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mockrepository.NewMockScopeRepository(ctrl)

	s := NewScopeService(mockRepo)

	ctx := context.Background()
	testIDs := []string{"scope-1", "scope-2"}
	expectedScopes := []model.Scope{{ID: "scope-1"}, {ID: "scope-2"}}
	repoError := errors.New("database query failed")

	testCases := []struct {
		name           string
		inputIDs       []string
		setupMock      func()
		expectedResult []model.Scope
		expectedError  error
	}{
		{
			name:     "Success - Should return scopes from repository",
			inputIDs: testIDs,
			setupMock: func() {
				mockRepo.EXPECT().
					GetScopesListByIDs(ctx, testIDs).
					Return(expectedScopes, nil)
			},
			expectedResult: expectedScopes,
			expectedError:  nil,
		},
		{
			name:     "Failure - Should return error when repository fails",
			inputIDs: testIDs,
			setupMock: func() {
				mockRepo.EXPECT().
					GetScopesListByIDs(ctx, testIDs).
					Return(nil, repoError)
			},
			expectedResult: nil,
			expectedError:  repoError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMock()
			scopes, err := s.GetScopesByIDs(ctx, tc.inputIDs)
			assert.Equal(t, tc.expectedResult, scopes)
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
