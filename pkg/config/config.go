package config

import (
	"flag"
	"fmt"
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
}
