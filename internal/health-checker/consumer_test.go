package health_checker

import (
	"VCS_SMS_Microservice/pkg/infra"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestConsumer_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReader := infra.NewMockKafkaReader(ctrl)
	mockReader.EXPECT().Close().Return(nil).Times(1)

	c := &consumer{
		kafkaReader: mockReader,
	}

	c.Stop()
}

func TestConsumer_performHealthCheck(t *testing.T) {

	testCases := []struct {
		name          string
		setupMocks    func(mockClient *MockServerClient, mockWriter *infra.MockKafkaWriter)
		expectedError bool
	}{
		{
			name: "Success Server Healthy (200 OK)",
			setupMocks: func(mockClient *MockServerClient, mockWriter *infra.MockKafkaWriter) {
				mockClient.EXPECT().GetServerHealthCheck(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(HealthCheckResponse{
					StatusCode: 200,
					Timestamp:  time.Now(),
				}, nil)
				mockWriter.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: false,
		},
		{
			name: "Success Server Inactive (Connection Refused)",
			setupMocks: func(mockClient *MockServerClient, mockWriter *infra.MockKafkaWriter) {
				mockClient.EXPECT().GetServerHealthCheck(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(HealthCheckResponse{
					Error: syscall.ECONNREFUSED,
				}, nil)
				mockWriter.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: false,
		},
		{
			name: "Success Server Network Error",
			setupMocks: func(mockClient *MockServerClient, mockWriter *infra.MockKafkaWriter) {
				mockClient.EXPECT().GetServerHealthCheck(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(HealthCheckResponse{
					Error: errors.New("some network error"),
				}, nil)
				mockWriter.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: false,
		},
		{
			name: "Success Server Configuration Error (404 Not Found)",
			setupMocks: func(mockClient *MockServerClient, mockWriter *infra.MockKafkaWriter) {
				mockClient.EXPECT().GetServerHealthCheck(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(HealthCheckResponse{
					StatusCode: 404,
					Timestamp:  time.Now(),
				}, nil)
				mockWriter.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: false,
		},
		{
			name: "Success Server Unhealthy (503 Service Unavailable)",
			setupMocks: func(mockClient *MockServerClient, mockWriter *infra.MockKafkaWriter) {
				mockClient.EXPECT().GetServerHealthCheck(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(HealthCheckResponse{
					StatusCode: 503,
					Timestamp:  time.Now(),
				}, nil)
				mockWriter.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: false,
		},
		{
			name: "Failure GetServerHealthCheck returns error",
			setupMocks: func(mockClient *MockServerClient, mockWriter *infra.MockKafkaWriter) {
				mockClient.EXPECT().GetServerHealthCheck(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(HealthCheckResponse{}, errors.New("client failed"))
			},
			expectedError: true,
		},
		{
			name: "Failure KafkaWriter WriteMessages returns error",
			setupMocks: func(mockClient *MockServerClient, mockWriter *infra.MockKafkaWriter) {
				mockClient.EXPECT().GetServerHealthCheck(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(HealthCheckResponse{
					StatusCode: 200,
					Timestamp:  time.Now(),
				}, nil)
				mockWriter.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).Return(errors.New("kafka write failed"))
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := NewMockServerClient(ctrl)
			mockWriter := infra.NewMockKafkaWriter(ctrl)

			tc.setupMocks(mockClient, mockWriter)

			c := &consumer{
				serverClient: mockClient,
				kafkaWriter:  mockWriter,
				logger:       zap.NewNop(),
			}

			err := c.performHealthCheck(context.Background(), "server-1", "127.0.0.1", 8080, 30, "/health")

			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConsumer_Start(t *testing.T) {
	validEvent := serverEvent{
		ID:                  "server-1",
		Ipv4:                "127.0.0.1",
		Port:                8080,
		HealthEndpoint:      "/health",
		HealthCheckInterval: 30,
	}
	validMessageValue, _ := json.Marshal(validEvent)
	validMessage := kafka.Message{Value: validMessageValue}

	testCases := []struct {
		name       string
		setupMocks func(mockReader *infra.MockKafkaReader, mockWriter *infra.MockKafkaWriter, mockClient *MockServerClient, wg *sync.WaitGroup)
	}{
		{
			name: "Success Process message successfully",
			setupMocks: func(mockReader *infra.MockKafkaReader, mockWriter *infra.MockKafkaWriter, mockClient *MockServerClient, wg *sync.WaitGroup) {
				gomock.InOrder(
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(validMessage, nil),
					mockClient.EXPECT().GetServerHealthCheck(gomock.Any(), validEvent.Ipv4, validEvent.Port, validEvent.HealthEndpoint).Return(HealthCheckResponse{StatusCode: 200}, nil),
					mockWriter.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).Return(nil),
					mockReader.EXPECT().CommitMessages(gomock.Any(), validMessage).Return(nil),
					mockReader.EXPECT().FetchMessage(gomock.Any()).DoAndReturn(func(_ context.Context) (kafka.Message, error) {
						wg.Done()
						return kafka.Message{}, io.EOF
					}),
				)
			},
		},
		{
			name: "Error FetchMessage returns an error",
			setupMocks: func(mockReader *infra.MockKafkaReader, mockWriter *infra.MockKafkaWriter, mockClient *MockServerClient, wg *sync.WaitGroup) {
				gomock.InOrder(
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(kafka.Message{}, errors.New("kafka connection error")),
					mockReader.EXPECT().FetchMessage(gomock.Any()).DoAndReturn(func(_ context.Context) (kafka.Message, error) {
						wg.Done()
						return kafka.Message{}, io.EOF
					}),
				)
			},
		},
		{
			name: "Skip Message value is nil",
			setupMocks: func(mockReader *infra.MockKafkaReader, mockWriter *infra.MockKafkaWriter, mockClient *MockServerClient, wg *sync.WaitGroup) {
				nilMessage := kafka.Message{Value: nil}
				gomock.InOrder(
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(nilMessage, nil),
					mockReader.EXPECT().CommitMessages(gomock.Any(), nilMessage).Return(nil),
					mockReader.EXPECT().FetchMessage(gomock.Any()).DoAndReturn(func(_ context.Context) (kafka.Message, error) {
						wg.Done()
						return kafka.Message{}, io.EOF
					}),
				)
			},
		},
		{
			name: "Error JSON unmarshal fails",
			setupMocks: func(mockReader *infra.MockKafkaReader, mockWriter *infra.MockKafkaWriter, mockClient *MockServerClient, wg *sync.WaitGroup) {
				invalidJSONMessage := kafka.Message{Value: []byte("this is not json")}
				gomock.InOrder(
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(invalidJSONMessage, nil),
					mockReader.EXPECT().CommitMessages(gomock.Any(), invalidJSONMessage).Return(nil),
					mockReader.EXPECT().FetchMessage(gomock.Any()).DoAndReturn(func(_ context.Context) (kafka.Message, error) {
						wg.Done()
						return kafka.Message{}, io.EOF
					}),
				)
			},
		},
		{
			name: "Error performHealthCheck fails",
			setupMocks: func(mockReader *infra.MockKafkaReader, mockWriter *infra.MockKafkaWriter, mockClient *MockServerClient, wg *sync.WaitGroup) {
				gomock.InOrder(
					mockReader.EXPECT().FetchMessage(gomock.Any()).Return(validMessage, nil),
					mockClient.EXPECT().GetServerHealthCheck(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(HealthCheckResponse{}, errors.New("health check failed")),
					mockReader.EXPECT().FetchMessage(gomock.Any()).DoAndReturn(func(_ context.Context) (kafka.Message, error) {
						wg.Done()
						return kafka.Message{}, io.EOF
					}),
				)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockReader := infra.NewMockKafkaReader(ctrl)
			mockWriter := infra.NewMockKafkaWriter(ctrl)
			mockClient := NewMockServerClient(ctrl)
			logger := zap.NewNop()
			c := NewConsumer(mockReader, mockWriter, mockClient, logger)
			var wg sync.WaitGroup
			wg.Add(1)
			tc.setupMocks(mockReader, mockWriter, mockClient, &wg)
			c.Start()
			wg.Wait()
		})
	}
}
