package app

import (
	"context"
	"net/http"
	_ "net/http/pprof"

	"github.com/NailUsmanov/practicum-shortener-url/internal/handlers"
	"github.com/NailUsmanov/practicum-shortener-url/internal/middleware"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/NailUsmanov/practicum-shortener-url/internal/tasks"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

type App struct {
	router     *chi.Mux
	storage    storage.Storage
	baseURL    string
	sugar      *zap.SugaredLogger
	deleteChan chan tasks.DeleteTask
}

func NewApp(s storage.Storage, baseURL string, sugar *zap.SugaredLogger) *App {
	r := chi.NewRouter()
	app := &App{
		router:     r, //разыменовываем указатель
		storage:    s,
		baseURL:    baseURL,
		sugar:      sugar,
		deleteChan: make(chan tasks.DeleteTask, 1000),
	}
	app.setupRoutes()
	return app
}

func (a *App) setupRoutes() {

	go func() {
		for task := range a.deleteChan {
			a.storage.MarkAsDeleted(context.Background(), task.ShortURLs, task.UserID)
		}
	}()
	// MiddleWare
	a.router.Use(middleware.LoggingMiddleWare(a.sugar))
	a.router.Use(middleware.AuthMiddleware)
	a.router.Use(middleware.GzipMiddleware)

	a.router.Post("/", handlers.NewCreateShortURL(a.storage, a.baseURL, a.sugar))
	a.router.Get("/{id}", handlers.NewRedirect(a.storage, a.sugar))
	a.router.Get("/ping", handlers.NewPingHandler(a.storage, a.sugar))

	a.router.Post("/api/shorten", handlers.NewCreateShortURLJSON(a.storage, a.baseURL, a.sugar))
	a.router.Post("/api/shorten/batch", handlers.NewCreateBatchJSON(a.storage, a.baseURL, a.sugar))
	a.router.Get("/api/user/urls", handlers.GetUserURLS(a.storage, a.baseURL, a.sugar))
	a.router.Delete("/api/user/urls", handlers.DeleteHandler(a.storage, a.sugar, a.deleteChan))
}

func (a *App) Run(addr string) error {
	return http.ListenAndServe(addr, a.router) //передаем указатель на роутер
}
