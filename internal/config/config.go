package config

import (
	"flag"
)

var FlagRunAddr string
var BaseURL string

func ParseFlag() {
	//для обозначения порта
	flag.StringVar(&FlagRunAddr, "a", ":8080", "address and port to run server")

	//для исходной ссылки
	flag.StringVar(&BaseURL, "b", "localhost:8080", "base URL for short links")

	flag.Parse()
}
