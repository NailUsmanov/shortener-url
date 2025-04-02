package config

import (
	"flag"
	"fmt"
	_ "os"
	"strings"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	RunAddr string `env:"SERVER_ADDRESS" envDefault:":8080"`
	BaseURL string `env:"BASE_URL"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.RunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&cfg.BaseURL, "b", "", "base URL for short links")
	flag.Parse()

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse env: %w", err)
	}

	if cfg.RunAddr == "" {
		cfg.RunAddr = ":8080" // Полный дефолт
	} else if !strings.Contains(cfg.RunAddr, ":") {
		cfg.RunAddr = ":" + cfg.RunAddr // Добавляем двоеточие если его нет
	}

	if cfg.BaseURL == "" {
		hostPort := cfg.RunAddr
		if hostPort == ":" {
			hostPort = ":8080" // Явно обрабатываем случай только ":"
		}
		if strings.HasPrefix(hostPort, ":") {
			hostPort = "localhost" + hostPort
		}
		cfg.BaseURL = fmt.Sprintf("http://%s", hostPort)
	}

	return cfg, nil
}
