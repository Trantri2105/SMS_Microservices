package config

import (
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type AppConfig struct {
	Server        ServerConfig
	Postgres      PostgresConfig
	Elasticsearch ElasticsearchConfig
	Mail          MailConfig
}

type MailConfig struct {
	Email            string `envconfig:"MAIL_EMAIL" required:"true"`
	Password         string `envconfig:"MAIL_PASSWORD" required:"true"`
	Host             string `envconfig:"MAIL_HOST" required:"true"`
	Port             int    `envconfig:"MAIL_PORT" required:"true"`
	AdminMailAddress string `envconfig:"MAIL_ADMIN_EMAIL" required:"true"`
}

type ServerConfig struct {
	Port                     string        `envconfig:"SERVER_PORT" default:"8080"`
	LogLevel                 string        `envconfig:"LOG_LEVEL" default:"info"`
	HealthCheckTaskQueueSize int           `envconfig:"HEALTH_CHECK_TASK_QUEUE_SIZE" default:"500"`
	HealthCheckWorker        int           `envconfig:"HEALTH_CHECK_WORKER" default:"100"`
	UserSessionTTL           time.Duration `envconfig:"USER_SESSION_TTL" default:"720h"`
}

type PostgresConfig struct {
	Host     string `envconfig:"POSTGRES_HOST" required:"true"`
	Port     int    `envconfig:"POSTGRES_PORT" required:"true"`
	User     string `envconfig:"POSTGRES_USER" required:"true"`
	Password string `envconfig:"POSTGRES_PASSWORD" required:"true"`
	DBName   string `envconfig:"POSTGRES_DB" required:"true"`
}

type ElasticsearchConfig struct {
	Addresses []string `envconfig:"ELASTICSEARCH_ADDRESSES" required:"true"`
}

func LoadConfig(path string) (AppConfig, error) {
	_ = godotenv.Load(path)

	var cfg AppConfig
	err := envconfig.Process("", &cfg)
	return cfg, err
}
