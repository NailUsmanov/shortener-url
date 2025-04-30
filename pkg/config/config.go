package config

import (
	"flag"
	"fmt"
	"strings"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	RunAddr    string `env:"SERVER_ADDRESS" envDefault:":8080"`
	BaseURL    string `env:"BASE_URL"`
	SaveInFile string `env:"FILE_STORAGE_PATH"`
	DataBase   string `env:"DATABASE_DSN"`
}

var (
	flagRunAddr    = flag.String("a", "", "address and port to run server")
	flagBaseURL    = flag.String("b", "", "base URL for short links")
	flagSaveInFile = flag.String("f", "", "if want to save short URL in file")
	flagDataBase   = flag.String("d", "", "if want to save short URL in DataBase")
)

func NewConfig() (*Config, error) {
	flag.Parse()
	cfg := &Config{}

	// Парсим переменные окружения
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse env: %w", err)
	}

	// Если флаг передан, перезаписываем значения
	if *flagRunAddr != "" {
		cfg.RunAddr = *flagRunAddr
	}
	if *flagBaseURL != "" {
		cfg.BaseURL = *flagBaseURL
	}
	if *flagSaveInFile != "" {
		cfg.SaveInFile = *flagSaveInFile
	}

	if *flagDataBase != "" {
		cfg.DataBase = *flagDataBase
	}

	// Устанавливаем значение по умолчанию
	if cfg.RunAddr == "" {
		cfg.RunAddr = ":8080"
	} else if !strings.Contains(cfg.RunAddr, ":") {
		cfg.RunAddr = ":" + cfg.RunAddr
	}

	if cfg.BaseURL == "" {
		hostPort := cfg.RunAddr
		if hostPort == ":" {
			hostPort = ":8080"
		}
		if strings.HasPrefix(hostPort, ":") {
			hostPort = "localhost" + hostPort
		}
		cfg.BaseURL = fmt.Sprintf("http://%s", hostPort)
	}

	return cfg, nil
}
