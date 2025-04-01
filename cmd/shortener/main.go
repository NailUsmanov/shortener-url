package main

import (
	"github.com/NailUsmanov/practicum-shortener-url/internal/app"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/NailUsmanov/practicum-shortener-url/pkg/config"
)

func main() {

	config.ParseFlag()
	store := storage.NewMemoryStorage()

	application := app.NewApp(store, config.BaseURL)

	if err := application.Run(config.FlagRunAddr); err != nil {
		panic(err)
	}
}
