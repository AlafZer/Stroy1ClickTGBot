package main

import (
	telegram "Stroy1ClickBot/bot/tgWorker"
	order "Stroy1ClickBot/order/orderReceiver"
	"Stroy1ClickBot/storage"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

type SetWebhookRequest struct {
	URL string `json:"url"`
}

type SetWebhookResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

var (
	tgToken      string
	pathToSQLite string
)

func main() {
	errCh := make(chan error, 2)
	var wg sync.WaitGroup

	// init of our variables and prepare our database
	//initAllStaticVars()

	db, err := storage.OpenSQLite(context.Background(), storage.OpenOptions{
		//Path: pathToSQLite,
		Path: "storage/data/notification.db",
	})
	if err != nil {
		log.Fatal("Cannot start the application because connection to SQLite failed")
	}
	defer func() {
		err = db.Close()
		if err != nil {
			log.Fatal("Cannot close the database connection")
		}
	}()

	err = storage.Migrate(context.Background(), db)
	if err != nil {
		log.Fatal("Cannot start the application because migration of SQLite failed:", err)
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
	tgToken = "8520034678:AAHpCgUOmOH96WwP3WT27xcqCdSuFainXLI"
	tgWorker := telegram.New(store, tgToken)
	//mustSetWebhook(tgToken)

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

func initAllStaticVars() {
	tgToken = strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	pathToSQLite = strings.TrimSpace(os.Getenv("SQLITE_PATH"))

	if tgToken == "" || pathToSQLite == "" {
		log.Fatal("tgToken or pathToSQLite environment variables is not specified")
	}
}

func mustSetWebhook(token string) {
	var payload SetWebhookRequest
	targetUrl := "https://tg-notification.stroy1click.com/api/v1/telegram/updates"

	payload.URL = targetUrl

	entry, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/setWebhook", token)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(entry))
	if err != nil {
		log.Fatal("Cannot start the application:", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if resp == nil {
		log.Fatal("Cannot start the application because response cannot be nil")
	}

	defer resp.Body.Close()

	if err != nil {
		log.Fatal("Cannot start the application:", err)
	}

	var data SetWebhookResponse

	dec := json.NewDecoder(resp.Body)

	err = dec.Decode(&data)
	if err != nil {
		log.Println(err)
	}

	log.Println(resp.StatusCode, data)
}
