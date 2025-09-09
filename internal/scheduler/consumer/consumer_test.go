package consumer

import (
	mockrepository "VCS_SMS_Microservice/internal/scheduler/mock/repository"
	"VCS_SMS_Microservice/internal/scheduler/model"
	"VCS_SMS_Microservice/pkg/infra"
	"context"
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

func newKafkaMessage(op string, id string) kafka.Message {
	event := serverEvent{
		Payload: struct {
			Op     string `json:"op"`
			Before struct {
				Id string `json:"id"`
			} `json:"before"`
			After struct {
				Id                  string `json:"id"`
				Ipv4                string `json:"ipv4"`
				Port                int    `json:"port"`
				HealthCheckInterval int    `json:"health_check_interval"`
				HealthEndpoint      string `json:"health_endpoint"`
			} `json:"after"`
		}{
			Op: op,
			Before: struct {
				Id string `json:"id"`
			}{Id: id},
			After: struct {
				Id                  string `json:"id"`
				Ipv4                string `json:"ipv4"`
				Port                int    `json:"port"`
				HealthCheckInterval int    `json:"health_check_interval"`
				HealthEndpoint      string `json:"health_endpoint"`
			}{
				Id:                  id,
				Ipv4:                "127.0.0.1",
				Port:                8080,
				HealthCheckInterval: 30,
				HealthEndpoint:      "/health",
			},
		},
	}
	val, _ := json.Marshal(event)
	return kafka.Message{Value: val}
}

func TestServerConsumer_Start(t *testing.T) {
	mockServer := model.Server{
		ID:                  "server-123",
		Ipv4:                "127.0.0.1",
		Port:                8080,
		HealthEndpoint:      "/health",
		HealthCheckInterval: 30,
	}
	createMessage := newKafkaMessage("c", mockServer.ID)
	updateMessage := newKafkaMessage("u", mockServer.ID)
	deleteMessage := newKafkaMessage("d", mockServer.ID)
	unknownMessage := newKafkaMessage("x", mockServer.ID)

	testCases := []struct {
		name       string
		setupMocks func(mockKafka *infra.MockKafkaReader, mockRepo *mockrepository.MockServerRepository)
	}{
		{
			name: "Success Handle Create Event",
			setupMocks: func(mockKafka *infra.MockKafkaReader, mockRepo *mockrepository.MockServerRepository) {
				gomock.InOrder(
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(createMessage, nil),
					mockRepo.EXPECT().CreateServer(gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ context.Context, s model.Server) (model.Server, error) {
							assert.Equal(t, mockServer.ID, s.ID)
							assert.Equal(t, mockServer.Ipv4, s.Ipv4)
							return s, nil
						},
					),
					mockKafka.EXPECT().CommitMessages(gomock.Any(), createMessage).Return(nil),
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Success Handle Update Event",
			setupMocks: func(mockKafka *infra.MockKafkaReader, mockRepo *mockrepository.MockServerRepository) {
				gomock.InOrder(
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(updateMessage, nil),
					mockRepo.EXPECT().UpdateServer(gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ context.Context, s model.Server) (model.Server, error) {
							assert.Equal(t, mockServer.ID, s.ID)
							assert.Equal(t, mockServer.Ipv4, s.Ipv4)
							return s, nil
						},
					),
					mockKafka.EXPECT().CommitMessages(gomock.Any(), updateMessage).Return(nil),
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Success Handle Delete Event",
			setupMocks: func(mockKafka *infra.MockKafkaReader, mockRepo *mockrepository.MockServerRepository) {
				gomock.InOrder(
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(deleteMessage, nil),
					mockRepo.EXPECT().DeleteServerById(gomock.Any(), mockServer.ID).Return(nil),
					mockKafka.EXPECT().CommitMessages(gomock.Any(), deleteMessage).Return(nil),
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Success Handle Unknown Event",
			setupMocks: func(mockKafka *infra.MockKafkaReader, mockRepo *mockrepository.MockServerRepository) {
				gomock.InOrder(
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(unknownMessage, nil),
					mockKafka.EXPECT().CommitMessages(gomock.Any(), unknownMessage).Return(nil),
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Failure FetchMessage returns error",
			setupMocks: func(mockKafka *infra.MockKafkaReader, mockRepo *mockrepository.MockServerRepository) {
				gomock.InOrder(
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, errors.New("kafka is down")),
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Skip Message value is nil",
			setupMocks: func(mockKafka *infra.MockKafkaReader, mockRepo *mockrepository.MockServerRepository) {
				gomock.InOrder(
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{Value: nil}, nil),
					mockKafka.EXPECT().CommitMessages(gomock.Any(), gomock.Any()).Return(nil),
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Failure - JSON Unmarshal error",
			setupMocks: func(mockKafka *infra.MockKafkaReader, mockRepo *mockrepository.MockServerRepository) {
				invalidMsg := kafka.Message{Value: []byte("this is not json")}
				gomock.InOrder(
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(invalidMsg, nil),
					mockKafka.EXPECT().CommitMessages(gomock.Any(), invalidMsg).Return(nil),
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Failure - JSON Unmarshal error and Commit error",
			setupMocks: func(mockKafka *infra.MockKafkaReader, mockRepo *mockrepository.MockServerRepository) {
				invalidMsg := kafka.Message{Value: []byte("this is not json")}
				gomock.InOrder(
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(invalidMsg, nil),
					mockKafka.EXPECT().CommitMessages(gomock.Any(), invalidMsg).Return(errors.New("commit failed")),
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Failure Repo CreateServer fails",
			setupMocks: func(mockKafka *infra.MockKafkaReader, mockRepo *mockrepository.MockServerRepository) {
				gomock.InOrder(
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(createMessage, nil),
					mockRepo.EXPECT().CreateServer(gomock.Any(), gomock.Any()).Return(model.Server{}, errors.New("database error")),
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Failure Repo UpdateServer fails",
			setupMocks: func(mockKafka *infra.MockKafkaReader, mockRepo *mockrepository.MockServerRepository) {
				gomock.InOrder(
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(updateMessage, nil),
					mockRepo.EXPECT().UpdateServer(gomock.Any(), gomock.Any()).Return(model.Server{}, errors.New("database error")),
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Failure Repo DeleteServerById fails",
			setupMocks: func(mockKafka *infra.MockKafkaReader, mockRepo *mockrepository.MockServerRepository) {
				gomock.InOrder(
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(deleteMessage, nil),
					mockRepo.EXPECT().DeleteServerById(gomock.Any(), mockServer.ID).Return(errors.New("database error")),
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
		{
			name: "Failure CommitMessages fails after successful repo operation",
			setupMocks: func(mockKafka *infra.MockKafkaReader, mockRepo *mockrepository.MockServerRepository) {
				gomock.InOrder(
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(createMessage, nil),
					mockRepo.EXPECT().CreateServer(gomock.Any(), gomock.Any()).Return(mockServer, nil),
					mockKafka.EXPECT().CommitMessages(gomock.Any(), createMessage).Return(errors.New("commit failed")),
					mockKafka.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, io.EOF),
				)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockKafkaReader := infra.NewMockKafkaReader(ctrl)
			mockServerRepo := mockrepository.NewMockServerRepository(ctrl)
			logger := zap.NewNop()
			tc.setupMocks(mockKafkaReader, mockServerRepo)
			consumer := NewServerConsumer(mockServerRepo, logger, mockKafkaReader)
			consumer.Start()
			time.Sleep(50 * time.Millisecond)
		})
	}
}

func TestServerConsumer_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockKafkaReader := infra.NewMockKafkaReader(ctrl)
	logger := zap.NewNop()
	mockKafkaReader.EXPECT().Close().Times(1)

	consumer := NewServerConsumer(nil, logger, mockKafkaReader)
	consumer.Stop()
}
