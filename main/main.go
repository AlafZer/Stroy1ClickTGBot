package main

import (
	telegram "Stroy1ClickBot/bot/tgWorker"
	order "Stroy1ClickBot/order/orderReceiver"
	"Stroy1ClickBot/storage"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type SetWebhookRequest struct {
	URL string `json:"url"`
}

func main() {
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	db, err := storage.OpenSQLite(context.Background(), storage.OpenOptions{
		Path: "./data/notification.db",
	})
	if err != nil {
		log.Fatal("Cannot start the application because connection to SQLite failed")
	}

	store := storage.NewStore(db)

	// starting orderReceiver
	ordReceiver := order.New(store)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Beginning of listening on the port 8080
	wg.Add(1)
	go func() {
		defer wg.Done()
		errCh <- ordReceiver.Listen()
	}()

	// starting tgWorker
	tgToken := mustToken()

	tgWorker := telegram.New(store, tgToken)
	mustSetWebhook()

	wg.Add(1)
	go func() {
		defer wg.Done()
		errCh <- tgWorker.ListenAndWork()
	}()

	// error handle
	select {
	case <-ctx.Done():
		ordReceiver.Shutdown()
		tgWorker.Shutdown()
	case err := <-errCh:
		ordReceiver.Shutdown()
		tgWorker.Shutdown()
		if err != nil {
			stop()
			log.Println("Error was capture from Listen() method of OrderReceiver or tgWorker type:", err)
		}
	}

	wg.Wait()
}

func mustToken() string {
	token := flag.String("tg-token", "", "token for access to telegram api")

	flag.Parse()

	if *token == "" {
		log.Fatal("token or bot name are not specified")
	}

	return *token
}

func mustSetWebhook() {
	var payload SetWebhookRequest
	targetUrl := "http://localhost:9090/api/v1/telegram/updates"

	payload.URL = targetUrl

	entry, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/setWebhook", telegram.TGToken)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(entry))
	if err != nil {
		log.Fatal("Cannot start the application:", err)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if resp == nil {
		log.Fatal("Cannot start the application because response cannot be nil")
	}

	defer resp.Body.Close()

	if err != nil {
		log.Fatal("Cannot start the application:", err)
	} else if resp.StatusCode != http.StatusOK {
		log.Fatal("Cannot start the application because response status code another then statusOK (200)", resp.StatusCode)
	}
}
