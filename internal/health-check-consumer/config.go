package health_check_consumer

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type AppConfig struct {
	Server   ServerConfig
	Postgres PostgresConfig
	Kafka    KafkaConfig
}

type ServerConfig struct {
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
}

type PostgresConfig struct {
	Host     string `envconfig:"POSTGRES_HOST" required:"true"`
	Port     int    `envconfig:"POSTGRES_PORT" required:"true"`
	User     string `envconfig:"POSTGRES_USER" required:"true"`
	Password string `envconfig:"POSTGRES_PASSWORD" required:"true"`
	DBName   string `envconfig:"POSTGRES_DB" required:"true"`
}

type KafkaConfig struct {
	Brokers         []string `envconfig:"KAFKA_BROKERS" required:"true"`
	ConsumerTopic   string   `envconfig:"KAFKA_CONSUMER_TOPIC" required:"true"`
	ConsumerGroupID string   `envconfig:"KAFKA_CONSUMER_GROUP_ID" required:"true"`
	ConsumerCnt     int      `envconfig:"KAFKA_CONSUMER_CNT" required:"true"`
}

func LoadConfig(path string) (AppConfig, error) {
	_ = godotenv.Load(path)

	var cfg AppConfig
	err := envconfig.Process("", &cfg)
	return cfg, err
}
