package telegram

import (
	"Stroy1ClickBot/storage"
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	webPort                        = "9090"
	minTimeDifferent time.Duration = 5 * time.Second
)

var TGToken string

type TGWorker struct {
	store  *storage.Store
	server *http.Server
}

type RequestResistor struct {
	mtx        sync.Mutex
	lastAppeal map[int64]time.Time
}

var rr *RequestResistor

func New(st *storage.Store, token string) *TGWorker {
	TGToken = token

	rr = &RequestResistor{}

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
	log.Println("Starting tgWorker ListenAndWork on port:", webPort)
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

func (rr *RequestResistor) checkLastAppeal(chatID int64) bool {
	rr.mtx.Lock()
	defer rr.mtx.Unlock()

	last, ok := rr.lastAppeal[chatID]

	if !ok {
		rr.lastAppeal[chatID] = time.Now()
		return true
	}

	dur := time.Now().Sub(last)

	if dur < minTimeDifferent {
		return false
	}

	rr.lastAppeal[chatID] = time.Now()
	return true
}
