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

	//Парсим переменные окружения, если есть
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse env: %w", err)
	}

	//Парсим флаги, если они переданы
	flagRunAddr := flag.String("a", "", "address and port to run server")
	flagBaseURL := flag.String("b", "", "base URL for short links")
	flag.Parse()

	//Если флаг передан, перезаписываем значения из переменных окружения
	if *flagRunAddr != "" {
		cfg.RunAddr = *flagRunAddr
	}
	if *flagBaseURL != "" {
		cfg.BaseURL = *flagBaseURL
	}

	//Устанавливаем значение по умолчанию, если ничего не задано
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
