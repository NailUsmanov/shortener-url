// Package config provides application configuration.
//
// Поддерживает загрузку параметров из переменных окружения и флагов командной строки.
// Включает настройки адреса сервера, базового URL, хранилища и ключей безопасности.
package config

import (
	"crypto/rand"
	"flag"
	"fmt"
	"strings"

	"github.com/caarlos0/env/v6"
)

// Config holds application configuration parameters.
//
// Включает адрес сервера, базовый URL, путь к файлу хранения, строку подключения к БД
// и секретный ключ для cookie.
type Config struct {
	RunAddr         string `env:"SERVER_ADDRESS" envDefault:":8080"`
	BaseURL         string `env:"BASE_URL"`
	SaveInFile      string `env:"FILE_STORAGE_PATH"`
	DataBase        string `env:"DATABASE_DSN"`
	CookieSecretKey []byte `env:"COOKIE_SECRET_KEY"`
}

var (
	flagRunAddr      = flag.String("a", "", "address and port to run server")
	flagBaseURL      = flag.String("b", "", "base URL for short links")
	flagSaveInFile   = flag.String("f", "", "if want to save short URL in file")
	flagDataBase     = flag.String("d", "", "if want to save short URL in DataBase")
	flagDataBaseLong = flag.String("database-dsn", "", "DSN to connect to the database")
)

// NewConfig загружает конфигурацию из переменных окружения и флагов.
//
// Значения из флагов имеют приоритет. Устанавливает значения по умолчанию
// и генерирует секретный ключ, если он не задан.
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

	if *flagDataBaseLong != "" {
		cfg.DataBase = *flagDataBaseLong
	} else if *flagDataBase != "" {
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

	// Генерируем ключ ТОЛЬКО если он не задан через ENV
	if len(cfg.CookieSecretKey) == 0 {
		cfg.CookieSecretKey = GenerateKeyToken()
	}

	return cfg, nil
}

// GenerateKeyToken generates a random 32-byte key for signing cookies.
func GenerateKeyToken() []byte {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	return key
}
