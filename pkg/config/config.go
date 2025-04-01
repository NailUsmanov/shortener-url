package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var FlagRunAddr string
var BaseURL string

func ParseFlag() {
	//для обозначения порта
	flag.StringVar(&FlagRunAddr, "a", ":8080", "address and port to run server")

	//для исходной ссылки
	flag.StringVar(&BaseURL, "b", "", "base URL for short links")

	flag.Parse()

	if BaseURL == "" {
		port := strings.TrimPrefix(FlagRunAddr, ":")
		if port == "" {
			port = "8080" // На случай если FlagRunAddr равен просто ":"
		}
		BaseURL = fmt.Sprintf("http://localhost:%s", port)
	}

	if EnvFlagRunAddr := os.Getenv("SERVER_ADDRESS"); EnvFlagRunAddr != "" {
		FlagRunAddr = EnvFlagRunAddr
	}

	if EnvBaseURL := os.Getenv("BASE_URL"); EnvBaseURL != "" {
		BaseURL = EnvBaseURL
	} else if BaseURL == "" {
		// Формируем URL по умолчанию только если не задан через флаг или env
		port := strings.TrimPrefix(FlagRunAddr, ":")
		if port == "" {
			port = "8080"
		}
		BaseURL = fmt.Sprintf("http://localhost:%s", port)
	}
}
