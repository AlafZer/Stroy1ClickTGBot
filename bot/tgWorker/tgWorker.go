package telegram

import (
	"Stroy1ClickBot/storage"
	"context"
	"log"
	"net/http"
	"time"
)

const (
	webPort = "9090"
)

var TGToken string

type TGWorker struct {
	store  *storage.Store
	server *http.Server
}

func New(st *storage.Store, token string) *TGWorker {
	TGToken = token

	tgWrkr := TGWorker{
		store: st,
	}

	tgWrkr.server = &http.Server{
		Addr:    ":" + webPort,
		Handler: tgWrkr.routes(),
	}

	return &tgWrkr
}

func (tgWrkr *TGWorker) ListenAndWork() error {
	return tgWrkr.server.ListenAndServe()
}

func (tgWrkr *TGWorker) Shutdown() {
	ctxT, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	// Graceful Shutdown
	if err := tgWrkr.server.Shutdown(ctxT); err != nil {
		log.Println("Failed to Shutdown server:", err)
	}
}
