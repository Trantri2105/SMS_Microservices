package config

import (
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type AppConfig struct {
	Server   ServerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	JWT      JWTConfig
}

type ServerConfig struct {
	Port           string        `envconfig:"SERVER_PORT" default:"8080"`
	LogLevel       string        `envconfig:"LOG_LEVEL" default:"info"`
	UserSessionTTL time.Duration `envconfig:"USER_SESSION_TTL" default:"720h"`
}

type PostgresConfig struct {
	Host     string `envconfig:"POSTGRES_HOST" required:"true"`
	Port     int    `envconfig:"POSTGRES_PORT" required:"true"`
	User     string `envconfig:"POSTGRES_USER" required:"true"`
	Password string `envconfig:"POSTGRES_PASSWORD" required:"true"`
	DBName   string `envconfig:"POSTGRES_DB" required:"true"`
}

type RedisConfig struct {
	Host string `envconfig:"REDIS_HOST" required:"true"`
	Port int    `envconfig:"REDIS_PORT" required:"true"`
}

type JWTConfig struct {
	SecretKey       string        `envconfig:"JWT_SECRET_KEY" required:"true"`
	AccessTokenTTL  time.Duration `envconfig:"JWT_ACCESS_TOKEN_TTL" default:"15m"`
	RefreshTokenTTL time.Duration `envconfig:"JWT_REFRESH_TOKEN_TTL" default:"168h"`
}

func LoadConfig(path string) (AppConfig, error) {
	_ = godotenv.Load(path)

	var cfg AppConfig
	err := envconfig.Process("", &cfg)
	return cfg, err
}
