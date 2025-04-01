package app

import (
	"net/http"

	"github.com/NailUsmanov/practicum-shortener-url/internal/handlers"
	"github.com/NailUsmanov/practicum-shortener-url/internal/storage"
	"github.com/go-chi/chi"
)

type App struct {
	router  chi.Mux
	storage storage.Storage
	handler *handlers.URLHandler
}

func NewApp(s storage.Storage, baseURL string) *App {
	r := chi.NewRouter()
	handler := handlers.NewURLHandler(s, baseURL)

	app := &App{
		router:  *r, //разыменовываем указатель
		storage: s,
		handler: handler,
	}

	app.setupRoutes()
	return app
}

func (a *App) setupRoutes() {
	a.router.Post("/", a.handler.CreateShortURL)
	a.router.Get("/{id}", a.handler.Redirect)
}

func (a *App) Run(addr string) error {
	return http.ListenAndServe(addr, &a.router) //передаем указатель на роутер
}
