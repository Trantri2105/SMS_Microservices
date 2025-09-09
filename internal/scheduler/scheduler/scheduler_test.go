package scheduler

import (
	mockrepository "VCS_SMS_Microservice/internal/scheduler/mock/repository"
	"VCS_SMS_Microservice/internal/scheduler/model"
	"VCS_SMS_Microservice/pkg/infra"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

var mockServers = []model.Server{
	{ID: "server-1", Ipv4: "1.1.1.1"},
	{ID: "server-2", Ipv4: "2.2.2.2"},
}

func TestServerScheduler_onTick(t *testing.T) {
	testCases := []struct {
		name       string
		setupMocks func(mockRepo *mockrepository.MockServerRepository, mockKafka *infra.MockKafkaWriter)
	}{
		{
			name: "Success Process servers successfully",
			setupMocks: func(mockRepo *mockrepository.MockServerRepository, mockKafka *infra.MockKafkaWriter) {
				gomock.InOrder(
					mockRepo.EXPECT().GetServersForHealthCheck(gomock.Any()).Return(mockServers, nil),
					mockKafka.EXPECT().WriteMessages(gomock.Any(), gomock.Len(2)).Return(nil),
					mockRepo.EXPECT().UpdateServersNextHealthCheckByIds(gomock.Any(), []string{"server-1", "server-2"}).Return(nil),
				)
			},
		},
		{
			name: "Failure GetServersForHealthCheck returns error",
			setupMocks: func(mockRepo *mockrepository.MockServerRepository, mockKafka *infra.MockKafkaWriter) {
				mockRepo.EXPECT().GetServersForHealthCheck(gomock.Any()).Return(nil, errors.New("db connection failed"))
			},
		},
		{
			name: "Success No servers to process",
			setupMocks: func(mockRepo *mockrepository.MockServerRepository, mockKafka *infra.MockKafkaWriter) {
				gomock.InOrder(
					mockRepo.EXPECT().GetServersForHealthCheck(gomock.Any()).Return([]model.Server{}, nil),
				)
			},
		},
		{
			name: "Failure - Kafka WriteMessages returns error",
			setupMocks: func(mockRepo *mockrepository.MockServerRepository, mockKafka *infra.MockKafkaWriter) {
				gomock.InOrder(
					mockRepo.EXPECT().GetServersForHealthCheck(gomock.Any()).Return(mockServers, nil),
					mockKafka.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).Return(errors.New("kafka is down")),
				)
			},
		},
		{
			name: "Failure - UpdateServersNextHealthCheckByIds returns error",
			setupMocks: func(mockRepo *mockrepository.MockServerRepository, mockKafka *infra.MockKafkaWriter) {
				gomock.InOrder(
					mockRepo.EXPECT().GetServersForHealthCheck(gomock.Any()).Return(mockServers, nil),
					mockKafka.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).Return(nil),
					mockRepo.EXPECT().UpdateServersNextHealthCheckByIds(gomock.Any(), gomock.Any()).Return(errors.New("failed to update timestamps")),
				)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mockrepository.NewMockServerRepository(ctrl)
			mockKafka := infra.NewMockKafkaWriter(ctrl)
			logger := zap.NewNop()

			tc.setupMocks(mockRepo, mockKafka)

			scheduler := &serverScheduler{
				logger:     logger,
				serverRepo: mockRepo,
				kafka:      mockKafka,
			}
			scheduler.onTick()
		})
	}
}

func TestServerScheduler_StartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockrepository.NewMockServerRepository(ctrl)
	mockKafka := infra.NewMockKafkaWriter(ctrl)
	logger := zap.NewNop()

	mockRepo.EXPECT().GetServersForHealthCheck(gomock.Any()).Return(mockServers, nil).MinTimes(1)
	mockKafka.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).Return(nil).MinTimes(1)
	mockRepo.EXPECT().UpdateServersNextHealthCheckByIds(gomock.Any(), gomock.Any()).Return(nil).MinTimes(1)

	mockKafka.EXPECT().Close().Times(1)

	scheduler := NewServerScheduler(logger, mockRepo, mockKafka)
	scheduler.Start()

	time.Sleep(1100 * time.Millisecond)

	scheduler.Stop()

	time.Sleep(50 * time.Millisecond)
}
