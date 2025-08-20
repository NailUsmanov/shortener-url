// Package app конфигурирует и запускает HTTP-приложение.
//
// Настраивает маршруты, middleware и запускает сервер.
package app

import (
	"context"
	"net"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/NailUsmanov/practicum-shortener-url/internal/handlers"
	"github.com/NailUsmanov/practicum-shortener-url/internal/middleware"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/NailUsmanov/practicum-shortener-url/internal/tasks"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

// App инкапсулирует конфигурацию HTTP-сервера.
//
// Включает маршрутизатор chi, хранилище, базовый URL, логгер и канал для фонового удаления URL.
type App struct {
	router     *chi.Mux
	storage    storage.Storage
	baseURL    string
	sugar      *zap.SugaredLogger
	deleteChan chan tasks.DeleteTask
	subnet     *net.IPNet
}

// NewApp создаёт и настраивает экземпляр App.
//
// Регистрирует маршруты и middleware.
func NewApp(s storage.Storage, baseURL string, sugar *zap.SugaredLogger, subnet *net.IPNet) *App {
	r := chi.NewRouter()
	app := &App{
		router:     r, //разыменовываем указатель
		storage:    s,
		baseURL:    baseURL,
		sugar:      sugar,
		deleteChan: make(chan tasks.DeleteTask, 1000),
		subnet:     subnet,
	}
	app.setupRoutes()
	return app
}

func (a *App) setupRoutes() {

	// MiddleWare
	a.router.Use(middleware.LoggingMiddleware(a.sugar))
	a.router.Use(middleware.AuthMiddleware)
	a.router.Use(middleware.GzipMiddleware)

	a.router.Post("/", handlers.NewCreateShortURL(a.storage, a.baseURL, a.sugar))
	a.router.Get("/{id}", handlers.NewRedirect(a.storage, a.sugar))
	a.router.Get("/ping", handlers.NewPingHandler(a.storage, a.sugar))

	a.router.Post("/api/shorten", handlers.NewCreateShortURLJSON(a.storage, a.baseURL, a.sugar))
	a.router.Post("/api/shorten/batch", handlers.NewCreateBatchJSON(a.storage, a.baseURL, a.sugar))
	a.router.Get("/api/user/urls", handlers.GetUserURLS(a.storage, a.baseURL, a.sugar))
	a.router.Delete("/api/user/urls", handlers.DeleteHandler(a.storage, a.sugar, a.deleteChan))
	a.router.Get("/api/internal/stats", handlers.GetStats(a.storage, a.subnet, a.sugar))
}

// Run запускает HTTP-сервер на указанном адресе.
func (a *App) Run(ctx context.Context, addr string) error {

	srv := http.Server{
		Addr:    addr,
		Handler: a.router,
	}
	go func() {

		for {
			select {
			case <-ctx.Done():
				return
			case task := <-a.deleteChan:
				a.storage.MarkAsDeleted(ctx, task.ShortURLs, task.UserID)
			}
		}
	}()

	go func() {
		<-ctx.Done()
		a.sugar.Infow("Shutting down server")
		sdCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sdCtx)
	}()

	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil // graceful путь
	}

	return err
}

// RunHTTPS запускает HTTPs-сервер на указанном адресе.
func (a *App) RunHTTPS(ctx context.Context, addr, certFile, keyFile string) error {
	srv := http.Server{
		Addr:    addr,
		Handler: a.router,
	}
	go func() {
		<-ctx.Done()
		a.sugar.Infof("Shutdown the server")
		sdCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sdCtx)
	}()

	err := srv.ListenAndServeTLS(certFile, keyFile)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
