package telegram

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (tgWrkr *TGWorker) routes() http.Handler {
	mux := chi.NewRouter()

	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)

	// This route is responsible by receive update requests from telegramAPI and linking userID to chatID
	mux.Post("/api/v1/telegram/updates", tgWrkr.tgUpdates)

	// This route is responsible by return telegram URL which will opened by user
	mux.Post("/api/v1/telegram/link", tgWrkr.tgLink)

	mux.Delete("/api/v1/telegram/binding/{UserID}", tgWrkr.tgDelete)

	return mux
}
