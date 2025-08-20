// Package config provides application configuration.
//
// Поддерживает загрузку параметров из переменных окружения и флагов командной строки.
// Включает настройки адреса сервера, базового URL, хранилища и ключей безопасности.
package config

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/env/v6"
)

// Config holds application configuration parameters.
//
// Включает адрес сервера защищенного и простого, базовый URL, путь к файлу хранения, строку подключения к БД
// и секретный ключ для cookie.
type Config struct {
	EnableHTTPS     bool   `env:"ENABLE_HTTPS" json:"enable_https"`
	CertFile        string `env:"TLS_CERT_FILE" json:"tls_cert_file"`
	KeyFile         string `env:"TLS_KEY_FILE" json:"tls_key_file"`
	RunAddr         string `env:"SERVER_ADDRESS" json:"server_address"`
	BaseURL         string `env:"BASE_URL" json:"base_url"`
	SaveInFile      string `env:"FILE_STORAGE_PATH" json:"file_storage_path"`
	DataBase        string `env:"DATABASE_DSN" json:"database_dsn"`
	CookieSecretKey []byte `env:"COOKIE_SECRET_KEY" json:"cookie_secret_key"`
	Config          string `env:"CONFIG"`
	TrustedSubnet   string `env:"TRUSTED_SUBNET"`
}

var (
	flagHTTPS         = flag.Bool("s", false, "if want to run server with TLS")
	flagCert          = flag.String("cert", "", "path to TLS cert file")
	flagKey           = flag.String("key", "", "path to TLS key file")
	flagRunAddr       = flag.String("a", "", "address and port to run server")
	flagBaseURL       = flag.String("b", "", "base URL for short links")
	flagSaveInFile    = flag.String("f", "", "if want to save short URL in file")
	flagDataBase      = flag.String("d", "", "if want to save short URL in DataBase")
	flagDataBaseLong  = flag.String("database-dsn", "", "DSN to connect to the database")
	flagCJSON         = flag.String("c", "", "config for the app")
	flagConfigJSON    = flag.String("config", "", "config for the app")
	flagTrustedSubnet = flag.String("t", "", "if you want to pass a string representation of classless addressing (CIDR)")
)

// NewConfig загружает конфигурацию из переменных окружения и флагов.
//
// Значения из флагов имеют приоритет. Устанавливает значения по умолчанию
// и генерирует секретный ключ, если он не задан.
func NewConfig() (*Config, error) {
	flag.Parse()
	cfg := &Config{}

	var path string

	switch {
	case *flagCJSON != "":
		path = *flagCJSON
	case *flagConfigJSON != "":
		path = *flagConfigJSON
	case os.Getenv("CONGIF") != "":
		path = os.Getenv("CONFIG")
	}

	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open config %q: %w", path, err)
		}
		defer file.Close()
		if err := json.NewDecoder(file).Decode(cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON")
		}
	}

	// Парсим переменные окружения
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse env: %w", err)
	}

	// Если флаг передан, перезаписываем значения
	if *flagHTTPS {
		cfg.EnableHTTPS = true
	}
	if *flagCert != "" {
		cfg.CertFile = *flagCert
	}
	if *flagKey != "" {
		cfg.KeyFile = *flagKey
	}
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

	if *flagTrustedSubnet != "" {
		cfg.TrustedSubnet = *flagTrustedSubnet
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

	// Если включён HTTPS и адрес по умолчанию (:8080), меняем на 443
	if cfg.EnableHTTPS && cfg.RunAddr == ":8080" {
		cfg.RunAddr = ":443"
	}
	if cfg.EnableHTTPS {
		if cfg.CertFile == "" {
			cfg.CertFile = "cert.pem"
		}
		if cfg.KeyFile == "" {
			cfg.KeyFile = "key.pem"
		}
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
