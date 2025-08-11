package health_checker

import (
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type AppConfig struct {
	Server ServerConfig
	Kafka  KafkaConfig
}

type ServerConfig struct {
	LogLevel       string        `envconfig:"LOG_LEVEL" default:"info"`
	MaxRetries     int           `envconfig:"MAX_RETRIES" default:"5"`
	InitialBackoff time.Duration `envconfig:"INITIAL_BACKOFF" default:"1s"`
	RequestTimeout time.Duration `envconfig:"REQUEST_TIMEOUT" default:"1s"`
}

type KafkaConfig struct {
	Brokers         []string `envconfig:"KAFKA_BROKERS" required:"true"`
	ConsumerTopic   string   `envconfig:"KAFKA_CONSUMER_TOPIC" required:"true"`
	ProducerTopic   string   `envconfig:"KAFKA_PRODUCER_TOPIC" required:"true"`
	ConsumerGroupID string   `envconfig:"KAFKA_CONSUMER_GROUP_ID" required:"true"`
	ConsumerCnt     int      `envconfig:"KAFKA_CONSUMER_CNT" required:"true"`
}

func LoadConfig(path string) (AppConfig, error) {
	_ = godotenv.Load(path)

	var cfg AppConfig
	err := envconfig.Process("", &cfg)
	return cfg, err
}
