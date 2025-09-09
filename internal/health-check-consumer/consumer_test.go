package health_check_consumer

import (
	mock_repository "VCS_SMS_Microservice/internal/server-service/mocks/repository"
	"VCS_SMS_Microservice/internal/server-service/model"
	"VCS_SMS_Microservice/pkg/infra"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func newKafkaMessage(t *testing.T, serverID, status string) kafka.Message {
	event := healthCheckEvent{
		ServerID: serverID,
		Status:   status,
	}
	value, err := json.Marshal(event)
	assert.NoError(t, err)
	return kafka.Message{Value: value}
}

func TestHealthCheckConsumer_Start(t *testing.T) {
	validMessage := newKafkaMessage(t, "server-001", "healthy")
	invalidJSONMessage := kafka.Message{Value: []byte("{not-a-json'")}
	nilValueMessage := kafka.Message{Value: nil}

	testCases := []struct {
		name       string
		setupMocks func(mockReader *infra.MockKafkaReader, mockRepo *mock_repository.MockServerRepository)
	}{
		{
			name: "Success Process valid message",
			setupMocks: func(mockReader *infra.MockKafkaReader, mockRepo *mock_repository.MockServerRepository) {
				gomock.InOrder(
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(validMessage, nil),
					mockRepo.EXPECT().UpdateServer(gomock.Any(), model.Server{ID: "server-001", Status: "healthy"}).Return(model.Server{}, nil),
					mockReader.EXPECT().CommitMessages(gomock.Any(), validMessage).Return(nil),
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Failure FetchMessage returns a generic error",
			setupMocks: func(mockReader *infra.MockKafkaReader, mockRepo *mock_repository.MockServerRepository) {
				gomock.InOrder(
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, errors.New("kafka broker unavailable")),
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Skip Message value is nil",
			setupMocks: func(mockReader *infra.MockKafkaReader, mockRepo *mock_repository.MockServerRepository) {
				gomock.InOrder(
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(nilValueMessage, nil),
					mockReader.EXPECT().CommitMessages(gomock.Any(), gomock.Any()).Return(nil),
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Failure JSON unmarshal fails and commit succeeds",
			setupMocks: func(mockReader *infra.MockKafkaReader, mockRepo *mock_repository.MockServerRepository) {
				gomock.InOrder(
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(invalidJSONMessage, nil),
					mockReader.EXPECT().CommitMessages(gomock.Any(), invalidJSONMessage).Return(nil),
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Failure JSON unmarshal fails and commit also fails",
			setupMocks: func(mockReader *infra.MockKafkaReader, mockRepo *mock_repository.MockServerRepository) {
				gomock.InOrder(
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(invalidJSONMessage, nil),
					mockReader.EXPECT().CommitMessages(gomock.Any(), invalidJSONMessage).Return(errors.New("failed to commit")),
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Failure UpdateServer returns an error",
			setupMocks: func(mockReader *infra.MockKafkaReader, mockRepo *mock_repository.MockServerRepository) {
				gomock.InOrder(
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(validMessage, nil),
					mockRepo.EXPECT().UpdateServer(gomock.Any(), gomock.Any()).Return(model.Server{}, errors.New("database timeout")),
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Failure CommitMessages fails after successful update",
			setupMocks: func(mockReader *infra.MockKafkaReader, mockRepo *mock_repository.MockServerRepository) {
				gomock.InOrder(
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(validMessage, nil),
					mockRepo.EXPECT().UpdateServer(gomock.Any(), gomock.Any()).Return(model.Server{}, nil),
					mockReader.EXPECT().CommitMessages(gomock.Any(), validMessage).Return(errors.New("failed to commit offset")),
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockReader := infra.NewMockKafkaReader(ctrl)
			mockRepo := mock_repository.NewMockServerRepository(ctrl)
			logger := zap.NewNop()

			tc.setupMocks(mockReader, mockRepo)

			consumer := NewHealthCheckConsumer(mockReader, mockRepo, logger)
			consumer.Start()

			time.Sleep(50 * time.Millisecond)
		})
	}
}

func TestHealthCheckConsumer_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReader := infra.NewMockKafkaReader(ctrl)
	logger := zap.NewNop()

	mockReader.EXPECT().Close().Times(1)

	consumer := NewHealthCheckConsumer(mockReader, nil, logger)
	consumer.Stop()
}
