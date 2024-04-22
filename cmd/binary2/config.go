package main

import (
	"github.com/caarlos0/env/v10"
)

type Config struct {
	NATSUrl string `env:"NATS_URL"`
}

func loadConfig() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
