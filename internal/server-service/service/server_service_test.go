package service

import (
	mockrepository "VCS_SMS_Microservice/internal/server-service/mocks/repository"
	"VCS_SMS_Microservice/internal/server-service/model"
	"VCS_SMS_Microservice/internal/server-service/repository"
	mockmail "VCS_SMS_Microservice/pkg/mail"

	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestServerService_GetServerUptimePercentage(t *testing.T) {
	ctx := context.Background()
	serverID := "server-123"
	startDate := time.Now().Add(-time.Hour)
	endDate := time.Now()

	testCases := []struct {
		name       string
		setupMocks func(healthCheckRepo *mockrepository.MockHealthCheckRepository)
		output     float64
		expectErr  bool
	}{
		{
			name: "Success Get uptime percentage",
			setupMocks: func(healthCheckRepo *mockrepository.MockHealthCheckRepository) {
				healthCheckRepo.EXPECT().
					GetServerUptimePercentage(ctx, serverID, startDate, endDate).
					Return(99.9, nil)
			},
			output:    99.9,
			expectErr: false,
		},
		{
			name: "Error Repository returns an error",
			setupMocks: func(healthCheckRepo *mockrepository.MockHealthCheckRepository) {
				healthCheckRepo.EXPECT().
					GetServerUptimePercentage(ctx, serverID, startDate, endDate).
					Return(0.0, errors.New("database error"))
			},
			output:    0.0,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockHealthCheckRepo := mockrepository.NewMockHealthCheckRepository(ctrl)
			tc.setupMocks(mockHealthCheckRepo)

			service := NewServerService(nil, mockHealthCheckRepo, nil)

			got, err := service.GetServerUptimePercentage(ctx, serverID, startDate, endDate)

			assert.Equal(t, tc.output, got)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServerService_ReportServersInformation(t *testing.T) {
	ctx := context.Background()
	startDate := time.Now().Add(-24 * time.Hour)
	endDate := time.Now()
	recipientMail := "test@example.com"

	serversInfo := repository.ServersHealthInformation{
		TotalServersCnt:         10,
		HealthyServersCnt:       8,
		UnhealthyServersCnt:     1,
		InactiveServersCnt:      1,
		AverageUptimePercentage: 95.5,
	}

	testCases := []struct {
		name       string
		setupMocks func(healthCheckRepo *mockrepository.MockHealthCheckRepository, mailSender *mockmail.MockSender)
		expectErr  bool
	}{
		{
			name: "Success Report sent successfully",
			setupMocks: func(healthCheckRepo *mockrepository.MockHealthCheckRepository, mailSender *mockmail.MockSender) {
				healthCheckRepo.EXPECT().
					GetAllServersHealthInformation(ctx, startDate, endDate).
					Return(serversInfo, nil)

				mailSender.EXPECT().
					SendMail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectErr: false,
		},
		{
			name: "Error Failed to get health information",
			setupMocks: func(healthCheckRepo *mockrepository.MockHealthCheckRepository, mailSender *mockmail.MockSender) {
				healthCheckRepo.EXPECT().
					GetAllServersHealthInformation(ctx, startDate, endDate).
					Return(repository.ServersHealthInformation{}, errors.New("db error"))
			},
			expectErr: true,
		},
		{
			name: "Error Failed to send mail",
			setupMocks: func(healthCheckRepo *mockrepository.MockHealthCheckRepository, mailSender *mockmail.MockSender) {
				healthCheckRepo.EXPECT().
					GetAllServersHealthInformation(ctx, startDate, endDate).
					Return(serversInfo, nil)

				mailSender.EXPECT().
					SendMail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("smtp error"))
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockHealthCheckRepo := mockrepository.NewMockHealthCheckRepository(ctrl)
			mockMailSender := mockmail.NewMockSender(ctrl)
			tc.setupMocks(mockHealthCheckRepo, mockMailSender)

			service := NewServerService(nil, mockHealthCheckRepo, mockMailSender)

			err := service.ReportServersInformation(ctx, startDate, endDate, recipientMail)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServerService_CreateServer(t *testing.T) {
	ctx := context.Background()
	serverToCreate := model.Server{ServerName: "NewServer"}

	expectedServerAfterCreate := model.Server{
		ID:         "new-id",
		ServerName: "NewServer",
		Status:     model.ServerStatusPending,
	}

	testCases := []struct {
		name       string
		setupMocks func(serverRepo *mockrepository.MockServerRepository)
		input      model.Server
		output     model.Server
		expectErr  bool
	}{
		{
			name:  "Success Server created",
			input: serverToCreate,
			setupMocks: func(serverRepo *mockrepository.MockServerRepository) {
				serverWithPendingStatus := serverToCreate
				serverWithPendingStatus.Status = model.ServerStatusPending

				serverRepo.EXPECT().
					CreateServer(ctx, serverWithPendingStatus).
					Return(expectedServerAfterCreate, nil)
			},
			output:    expectedServerAfterCreate,
			expectErr: false,
		},
		{
			name:  "Error Repository fails to create server",
			input: serverToCreate,
			setupMocks: func(serverRepo *mockrepository.MockServerRepository) {
				serverWithPendingStatus := serverToCreate
				serverWithPendingStatus.Status = model.ServerStatusPending

				serverRepo.EXPECT().
					CreateServer(ctx, serverWithPendingStatus).
					Return(model.Server{}, errors.New("db conflict"))
			},
			output:    serverToCreate,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockServerRepo := mockrepository.NewMockServerRepository(ctrl)
			tc.setupMocks(mockServerRepo)

			service := NewServerService(mockServerRepo, nil, nil)

			got, err := service.CreateServer(ctx, tc.input)

			if tc.expectErr {
				tc.output.Status = model.ServerStatusPending
			}
			assert.Equal(t, tc.output, got)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServerService_DeleteServer(t *testing.T) {
	ctx := context.Background()
	serverID := "server-to-delete"

	testCases := []struct {
		name       string
		setupMocks func(serverRepo *mockrepository.MockServerRepository)
		wantErr    bool
	}{
		{
			name: "Success Server deleted",
			setupMocks: func(serverRepo *mockrepository.MockServerRepository) {
				serverRepo.EXPECT().DeleteServerById(ctx, serverID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Error Server not found or db error",
			setupMocks: func(serverRepo *mockrepository.MockServerRepository) {
				serverRepo.EXPECT().DeleteServerById(ctx, serverID).Return(errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockServerRepo := mockrepository.NewMockServerRepository(ctrl)
			tc.setupMocks(mockServerRepo)

			service := NewServerService(mockServerRepo, nil, nil)

			err := service.DeleteServer(ctx, serverID)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServerService_GetServers(t *testing.T) {
	ctx := context.Background()

	serversList := []model.Server{
		{ID: "1", ServerName: "Server A", Status: "healthy"},
		{ID: "2", ServerName: "Server B", Status: "unhealthy"},
	}

	type args struct {
		serverName string
		status     string
		sortBy     string
		sortOrder  string
		limit      int
		offset     int
	}

	testCases := []struct {
		name       string
		args       args
		setupMocks func(serverRepo *mockrepository.MockServerRepository)
		output     []model.Server
		expectErr  bool
	}{
		{
			name: "Success Get servers with filters",
			args: args{serverName: "Server", status: "healthy", limit: 10, offset: 0},
			setupMocks: func(serverRepo *mockrepository.MockServerRepository) {
				serverRepo.EXPECT().
					GetServers(ctx, "Server", "healthy", "", "", 10, 0).
					Return(serversList, nil)
			},
			output:    serversList,
			expectErr: false,
		},
		{
			name: "Error Repository returns an error",
			args: args{limit: 10, offset: 0},
			setupMocks: func(serverRepo *mockrepository.MockServerRepository) {
				serverRepo.EXPECT().
					GetServers(ctx, "", "", "", "", 10, 0).
					Return(nil, errors.New("database connection lost"))
			},
			output:    nil,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockServerRepo := mockrepository.NewMockServerRepository(ctrl)
			tc.setupMocks(mockServerRepo)

			service := NewServerService(mockServerRepo, nil, nil)

			got, err := service.GetServers(ctx, tc.args.serverName, tc.args.status, tc.args.sortBy, tc.args.sortOrder, tc.args.limit, tc.args.offset)

			assert.Equal(t, tc.output, got)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServerService_ImportServers(t *testing.T) {
	ctx := context.Background()

	serversToImport := []model.Server{
		{ServerName: "Server C"},
		{ServerName: "Server D"},
	}

	expectedServersForRepo := []model.Server{
		{ServerName: "Server C", Status: model.ServerStatusPending},
		{ServerName: "Server D", Status: model.ServerStatusPending},
	}

	inserted := []model.Server{{ID: "3", ServerName: "Server C", Status: model.ServerStatusPending}}
	nonInserted := []model.Server{{ServerName: "Server D", Status: model.ServerStatusPending}}

	testCases := []struct {
		name              string
		setupMocks        func(serverRepo *mockrepository.MockServerRepository)
		outputInserted    []model.Server
		outputNonInserted []model.Server
		expectErr         bool
	}{
		{
			name: "Success Servers imported correctly",
			setupMocks: func(serverRepo *mockrepository.MockServerRepository) {
				serverRepo.EXPECT().
					ImportServers(ctx, expectedServersForRepo).
					Return(inserted, nonInserted, nil)
			},
			outputInserted:    inserted,
			outputNonInserted: nonInserted,
			expectErr:         false,
		},
		{
			name: "Error Repository returns an error",
			setupMocks: func(serverRepo *mockrepository.MockServerRepository) {
				serverRepo.EXPECT().
					ImportServers(ctx, expectedServersForRepo).
					Return(nil, nil, errors.New("transaction failed"))
			},
			outputInserted:    nil,
			outputNonInserted: nil,
			expectErr:         true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockServerRepo := mockrepository.NewMockServerRepository(ctrl)
			tc.setupMocks(mockServerRepo)

			service := NewServerService(mockServerRepo, nil, nil)

			gotInserted, gotNonInserted, err := service.ImportServers(ctx, serversToImport)

			assert.Equal(t, tc.outputInserted, gotInserted)
			assert.Equal(t, tc.outputNonInserted, gotNonInserted)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServerService_UpdateServer(t *testing.T) {
	ctx := context.Background()

	serverToUpdate := model.Server{
		ID:         "existing-id-1",
		ServerName: "Updated Name",
	}

	testCases := []struct {
		name       string
		setupMocks func(serverRepo *mockrepository.MockServerRepository)
		input      model.Server
		want       model.Server
		wantErr    bool
	}{
		{
			name:  "Success Server updated",
			input: serverToUpdate,
			setupMocks: func(serverRepo *mockrepository.MockServerRepository) {
				serverRepo.EXPECT().
					UpdateServer(ctx, serverToUpdate).
					Return(serverToUpdate, nil)
			},
			want:    serverToUpdate,
			wantErr: false,
		},
		{
			name:  "Error Repository fails to update",
			input: serverToUpdate,
			setupMocks: func(serverRepo *mockrepository.MockServerRepository) {
				serverRepo.EXPECT().
					UpdateServer(ctx, serverToUpdate).
					Return(model.Server{}, errors.New("server not found"))
			},
			want:    model.Server{},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockServerRepo := mockrepository.NewMockServerRepository(ctrl)
			tc.setupMocks(mockServerRepo)

			service := NewServerService(mockServerRepo, nil, nil)

			got, err := service.UpdateServer(ctx, tc.input)

			assert.Equal(t, tc.want, got)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
