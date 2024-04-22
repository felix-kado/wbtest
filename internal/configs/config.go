package configs

import (
	"fmt"

	"github.com/caarlos0/env/v10"
)

type Config struct { // Exported Config struct
	DBHost     string `env:"DB_HOST"`
	DBUser     string `env:"DB_USER"`
	DBPassword string `env:"DB_PASSWORD"`
	DBName     string `env:"DB_NAME"`
	NATSUrl    string `env:"NATS_URL"`
	NWorkers   int    `env:"NWORKERS"`
	ServerPort string `env:"SERVER_PORT"`
	ConnString string
}

func LoadConfig() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.NWorkers == 0 {
		cfg.NWorkers = 8
	}
	cfg.ConnString = fmt.Sprintf("user=%s dbname=%s sslmode=disable password=%s host=%s", cfg.DBUser, cfg.DBName, cfg.DBPassword, cfg.DBHost)
	return cfg, nil
}
