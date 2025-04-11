package app

import (
	"net/http"

	"github.com/NailUsmanov/practicum-shortener-url/internal/handlers"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

type Storage interface {
	Save(url string) (string, error)
	Get(key string) (string, error)
}

type App struct {
	router  *chi.Mux
	storage Storage
	handler *handlers.URLHandler
	sugar   *zap.SugaredLogger
}

func NewApp(s storage.Storage, baseURL string, sugar *zap.SugaredLogger) *App {
	r := chi.NewRouter()
	handler := handlers.NewURLHandler(s, baseURL, sugar)

	app := &App{
		router:  r, //разыменовываем указатель
		storage: s,
		handler: handler,
		sugar:   sugar,
	}

	app.setupRoutes()
	return app
}

func (a *App) setupRoutes() {
	a.router.Post("/", handlers.WithLogging(http.HandlerFunc(a.handler.CreateShortURL), a.sugar))
	a.router.Get("/{id}", handlers.WithLoggingRedirect(http.HandlerFunc(a.handler.Redirect), a.sugar))
	a.router.Post("/api/shorten", handlers.WithLogging(http.HandlerFunc(a.handler.CreateShortURLJSON), a.sugar))
}

func (a *App) Run(addr string) error {
	return http.ListenAndServe(addr, a.router) //передаем указатель на роутер
}
