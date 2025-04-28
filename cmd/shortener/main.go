package main

import (
	"log"

	"github.com/NailUsmanov/practicum-shortener-url/internal/app"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/NailUsmanov/practicum-shortener-url/pkg/config"
	"go.uber.org/zap"
)

func main() {

	// создаём предустановленный регистратор zap
	logger, err := zap.NewDevelopment()
	if err != nil {
		// вызываем панику, если ошибка
		panic(err)
	}
	defer logger.Sync()

	// делаем регистратор SugaredLogger
	sugar := logger.Sugar()

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var store storage.Storage
	if cfg.SaveInFile != "" {
		store = storage.NewFileStorage(cfg.SaveInFile)
		sugar.Info("Using file storage")
	} else {
		store = storage.NewMemoryStorage()
		sugar.Info("Using in-memory storage")
	}

	application := app.NewApp(store, cfg.BaseURL, sugar)

	if err := application.Run(cfg.RunAddr); err != nil {
		sugar.Fatalln(err)
	}
}
