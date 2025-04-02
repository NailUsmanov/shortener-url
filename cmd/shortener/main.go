package main

import (
	"log"

	"github.com/NailUsmanov/practicum-shortener-url/internal/app"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/NailUsmanov/practicum-shortener-url/pkg/config"
)

func main() {

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	store := storage.NewMemoryStorage()

	application := app.NewApp(store, cfg.BaseURL)

	if err := application.Run(cfg.RunAddr); err != nil {
		panic(err)
	}
}
