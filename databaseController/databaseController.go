package dbController

import (
	"Stroy1ClickBot/repository"
	"context"
	"fmt"
	"log"
	"net/http"
)

const port = 5431

type DatabaseController struct {
	DB     *repository.Store
	token  string
	server *http.Server
}

func NewDBController(storage *repository.Store, token string) *DatabaseController {
	var temp = &DatabaseController{
		DB:    storage,
		token: token,
	}

	temp.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: temp.routes(),
	}

	return temp
}

func (dbCtrl *DatabaseController) ListenAndServe(ctx context.Context) error {
	err := dbCtrl.server.ListenAndServe()
	return fmt.Errorf("error from DatabaseController: %w", err)
}

func (dbCtrl *DatabaseController) Shutdown(ctx context.Context) {
	err := dbCtrl.server.Shutdown(ctx)
	if err != nil {
		log.Printf("Failed Graceful Shutdown for database control: %s", err)
	}
}
