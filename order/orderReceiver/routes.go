package order

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (ordR *OrderReceiver) routes() http.Handler {
	mux := chi.NewRouter()

	mux.Use(middleware.Heartbeat("/ping"))

	mux.Post("/api/v1/telegram/send", ordR.receiveAndSend)

	return mux
}
