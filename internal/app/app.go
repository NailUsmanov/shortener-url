package app

import (
	"net/http"

	"github.com/NailUsmanov/practicum-shortener-url/internal/handlers"
	"github.com/NailUsmanov/practicum-shortener-url/internal/middleware"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

type App struct {
	router  *chi.Mux
	storage storage.Storage
	baseURL string
	sugar   *zap.SugaredLogger
}

func NewApp(s storage.Storage, baseURL string, sugar *zap.SugaredLogger) *App {
	r := chi.NewRouter()

	app := &App{
		router:  r, //разыменовываем указатель
		storage: s,
		baseURL: baseURL,
		sugar:   sugar,
	}

	app.setupRoutes()
	return app
}

func (a *App) setupRoutes() {
	// MiddleWare
	a.router.Use(middleware.GzipMiddleware)
	a.router.Use(middleware.LoggingMiddleWare(a.sugar))
	a.router.Use(middleware.AuthMiddleWare)

	// POST /api/shorten/batch
	a.router.Post("/api/shorten/batch", handlers.NewCreateBatchJSON(a.storage, a.baseURL, a.sugar))

	// POST /api/shorten
	a.router.Post("/api/shorten", handlers.NewCreateShortURLJSON(a.storage, a.baseURL, a.sugar))

	// POST
	a.router.Post("/", handlers.NewCreateShortURL(a.storage, a.baseURL, a.sugar))

	// GET
	a.router.Get("/{id}", handlers.NewRedirect(a.storage, a.sugar))

	// GET PING
	a.router.Get("/ping", handlers.NewPingHandler(a.storage, a.sugar))

	// GET /api/user/urls
	a.router.Get("/api/user/urls", handlers.GetUserURLS(a.storage, a.baseURL, a.sugar))
}

func (a *App) Run(addr string) error {
	return http.ListenAndServe(addr, a.router) //передаем указатель на роутер
}
