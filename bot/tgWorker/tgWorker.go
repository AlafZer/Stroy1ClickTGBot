package telegram

import (
	"Stroy1ClickBot/storage"
	"net/http"
)

const (
	webPort = "9090"
)

var TGToken string
var botName string

type TGWorker struct {
	store  *storage.Store
	server *http.Server
}

func New(st *storage.Store, token, btName string) *TGWorker {
	tgToken = token
	botName = btName

	return &TGWorker{
		store:  st,
		server: &http.Server{},
	}
}

func (tgWrkr *TGWorker) ListenAndWork() error {
	return tgWrkr.server.ListenAndServe()
}
