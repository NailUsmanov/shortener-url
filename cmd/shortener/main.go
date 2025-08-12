package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os/signal"
	"syscall"

	"github.com/NailUsmanov/practicum-shortener-url/internal/app"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/NailUsmanov/practicum-shortener-url/pkg/config"
	"go.uber.org/zap"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	// Закрываем соединение только для БД
	if dbStore, ok := store.(*storage.DataBaseStorage); ok {
		defer dbStore.Close()
	}
	// Составляем защищенное соединение
	if cfg.EnableHTTPS {
		err := application.RunHTTPS(ctx, cfg.RunAddr, cfg.CertFile, cfg.KeyFile)
		if err != nil && err != http.ErrServerClosed {
			sugar.Fatalln(err)
		}
	} else {
		err = application.Run(ctx, cfg.RunAddr)
		if err != nil {
			sugar.Fatalln(err)
		}
	}

}
