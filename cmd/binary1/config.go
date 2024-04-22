package main

import (
	"github.com/caarlos0/env/v10"
)

type Config struct {
	NATSUrl    string `env:"NATS_URL"`
	NWorkers   int    `env:"NWORKERS"`
	ServerPort string `env:"SERVER_PORT"`
	ConnString string `env:"DATABASE_URL"`
}

func LoadConfig() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.NWorkers == 0 {
		cfg.NWorkers = 8
	}
	return cfg, nil
}
