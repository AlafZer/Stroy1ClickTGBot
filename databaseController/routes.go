package dbController

import (
	"net/http"

	_ "Stroy1ClickBot/docs" // <-- это сгенерирует swag init

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (dbCtrl *DatabaseController) routes() http.Handler {
	mux := chi.NewRouter()

	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.Use(dbCtrl.auth)

	mux.Route("/api/v1", func(r chi.Router) {
		r.Route("/users", func(r chi.Router) {
			r.Get("/", dbCtrl.getAllUsersHandler)
			r.Get("/{UserID}", dbCtrl.getUserByUserIDHandler)
		})

		r.Get("/tokens", dbCtrl.getAllTokensHandler)
		r.Post("/admin", dbCtrl.setAdminRoleHandler)
	})

	return mux
}
