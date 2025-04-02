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

	if cfg.BaseURL == "" {
		port := strings.TrimPrefix(cfg.RunAddr, ":")
		if port == "" {
			port = "8080" // На случай если FlagRunAddr равен просто ":"
		}
		cfg.BaseURL = fmt.Sprintf("http://localhost:%s", port)
	}

	return cfg, nil
}
