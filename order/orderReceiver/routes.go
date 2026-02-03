package order

import (
	"net/http"

	_ "Stroy1ClickBot/docs" // <-- это сгенерирует swag init
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

func (ordR *OrderReceiver) routes() http.Handler {
	mux := chi.NewRouter()

	mux.Use(middleware.Heartbeat("/ping"))

	// This route is sending order messages to admin and user
	mux.Post("/api/v1/telegram/send", ordR.receiveAndSend)

	//HTML documentation
	mux.Get("/swagger/*", httpSwagger.WrapHandler)

	return mux
}
