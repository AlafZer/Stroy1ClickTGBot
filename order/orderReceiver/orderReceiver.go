package order

import (
	"Stroy1ClickBot/storage"
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

const (
	webPort = "8080"
)

type OrderReceiver struct {
	server *http.Server
	store  *storage.Store
}

func New(store *storage.Store) *OrderReceiver {
	ordR := &OrderReceiver{
		store: store,
	}

	srv := &http.Server{
		Addr:    ":" + webPort,
		Handler: ordR.routes(),
	}

	ordR.server = srv

	return ordR
}

func (ordR *OrderReceiver) Listen() error {
	log.Println("Starting OrderReceiverServer on port:", webPort)
	err := ordR.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (ordR *OrderReceiver) Shutdown() {
	ctxT, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	// Graceful Shutdown
	if err := ordR.server.Shutdown(ctxT); err != nil {
		log.Println("Failed to Shutdown server:", err)
	}
}
