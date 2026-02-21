package order

import (
	"Stroy1ClickBot/repository"
	"context"
	"log"
)

const (
	port = 8080
)

type OrderReceiver struct {
	//server *http.Server
	store    *repository.Store
	instance string
}

func New(store *repository.Store, instance string) *OrderReceiver {
	ordR := &OrderReceiver{
		store:    store,
		instance: instance,
	}

	return ordR
}

func (ordR *OrderReceiver) ListenAndServe(ctx context.Context) error {
	log.Printf("Starting OrderReceiverServer on port: :%d\n", port)
	err := ordR.receiveAndSend(ctx)

	return err
}

//func (ordR *OrderReceiver) Shutdown(ctx context.Context) {
//
//	// Graceful Shutdown
//	if err := ordR.server.Shutdown(ctx); err != nil {
//		log.Println("Failed to Shutdown server:", err)
//	}
//}
