package main

import (
	order "Stroy1ClickBot/order/orderReceiver"
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	// starting orderReceiver
	ordReceiver := order.New()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Beginning of listening on the port 8080
	wg.Add(1)
	go func() {
		defer wg.Done()
		errCh <- ordReceiver.Listen()
	}()

	// starting tgWorker

	// error handle
	select {
	case <-ctx.Done():
		ordReceiver.Shutdown()
	case err := <-errCh:
		ordReceiver.Shutdown()
		if err != nil {
			stop()
			log.Println("Error was capture from Listen() method of OrderServer type:", err)
		}
	}

	wg.Wait()
}

func mustTokenAndBotName() (string, string) {
	token := flag.String("tg-token", "", "token for access to telegram api")
	botName := flag.String("bot-name", "", "name of telegram bot")

	flag.Parse()

	if *token == "" || *botName == "" {
		log.Fatal("token or bot name are not specified")
	}

	return *token, *botName
}
