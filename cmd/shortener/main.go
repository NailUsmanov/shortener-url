package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/NailUsmanov/practicum-shortener-url/internal/app"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/NailUsmanov/practicum-shortener-url/pkg/config"
	"go.uber.org/zap"
)

func main() {
	go func() {
		log.Println("pprof listening on http://localhost:6060")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

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
		sugar.Infof("Using file storage at: %s", cfg.SaveInFile)
		store, err = storage.NewFileStorage(cfg.SaveInFile)
		if err != nil {
			sugar.Fatalf("failed to initialize file storage: %v", err)
		}
		sugar.Info("Using file storage")
	} else if cfg.DataBase != "" {
		store, err = storage.NewDataBaseStorage(cfg.DataBase)
		if err != nil {
			log.Fatalf("Failed to load DataBase: %v", err)
		}

	} else {
		store = storage.NewMemoryStorage()
		sugar.Info("Using in-memory storage")
	}

	application := app.NewApp(store, cfg.BaseURL, sugar)

	// Закрываем соединение только для БД
	if dbStore, ok := store.(*storage.DataBaseStorage); ok {
		defer dbStore.Close()
	}

	if err := application.Run(cfg.RunAddr); err != nil {
		sugar.Fatalln(err)
	}
}
