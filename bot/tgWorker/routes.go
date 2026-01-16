package telegram

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (tgWrkr *TGWorker) routes() http.Handler {
	mux := chi.NewRouter()

	mux.Post("/api/v1/telegram/send_updates", tgWrkr.tgUpdates)
	mux.Get("/api/v1/telegram/link", tgWrkr.tgLink)

	return mux
}
