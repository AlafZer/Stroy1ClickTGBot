package telegram

import (
	"Stroy1ClickBot/repository"
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	port                           = 9090
	minTimeDifferent time.Duration = 5 * time.Second
)

var (
	TGToken  string
	TokenAPI string
)

type TGWorker struct {
	store  *repository.Store
	server *http.Server
}

type RequestResistor struct {
	mtx        sync.Mutex
	lastAppeal map[int64]time.Time
}

var rr *RequestResistor

func New(st *repository.Store, token, tokenApi string) *TGWorker {
	TGToken = token
	TokenAPI = tokenApi

	rr = &RequestResistor{
		mtx:        sync.Mutex{},
		lastAppeal: make(map[int64]time.Time),
	}

	tgWrkr := TGWorker{
		store: st,
	}

	tgWrkr.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: tgWrkr.routes(),
	}

	return &tgWrkr
}

func (tgWrkr *TGWorker) ListenAndServe(ctx context.Context) error {
	log.Printf("Starting tgWorker ListenAndWork on port :%d", port)

	ctxC, canc := context.WithCancel(ctx)
	defer canc()

	clner := NewCleaner(tgWrkr.store, time.Hour)

	errCh := make(chan error)

	go func() {
		errCh <- clner.Clean(ctxC)
	}()

	// Starting ListenandServe and waiting for Graceful Shutdown
	err := tgWrkr.server.ListenAndServe()

	// Processing error from errCh
	canc()

	errFromCleaner := <-errCh
	if errFromCleaner != nil {
		log.Printf("An error occurred from cleaner: %s", errFromCleaner)
	}
	close(errCh)

	// returning error from server
	return fmt.Errorf("error from TGWorker: %w", err)
}

func (tgWrkr *TGWorker) Shutdown(ctx context.Context) {
	// Graceful Shutdown with context
	if err := tgWrkr.server.Shutdown(ctx); err != nil {
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

	dur := time.Since(last)

	if dur < minTimeDifferent {
		return false
	}

	rr.lastAppeal[chatID] = time.Now()
	return true
}
