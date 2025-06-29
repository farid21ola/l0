package config

import (
	"fmt"
	"l0/internal/broker"
	"l0/internal/repository"
	"l0/internal/transport/rest"
	"l0/pkg/logger"

	env "github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

type Config struct {
	DbConfig     repository.Config
	LoggerConfig logger.Config
	ServerConfig rest.Config
	KafkaConfig  broker.Config
}

func NewConfig() (*Config, error) {
	godotenv.Load(".env")

	cfg := Config{}

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	return &cfg, nil
}
