// @title           Stroy1Click Bot API
// @version         1.0
// @description     Internal APIs for order-service and telegram binding
// @BasePath        /
// @schemes         http https
package main

import (
	telegram "Stroy1ClickBot/bot/tgWorker"
	dbController "Stroy1ClickBot/databaseController"
	order "Stroy1ClickBot/order/orderReceiver"
	"Stroy1ClickBot/repository"
	"context"
	"log"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	tgToken      string
	pathToSQLite string
	//domain       string
	tgApiToken    string
	accessToken   string
	kafkaInstance string
)

type ServerInterface interface {
	ListenAndServe(ctx context.Context) error
}

func main() {
	var wg sync.WaitGroup

	// initialization of our variables and prepare our database
	initAllStaticVars()
	store := prepareDB(pathToSQLite)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// TODO Initialize servers, create error channel and
	ordReceiver := order.New(store, kafkaInstance)
	tgWorker := telegram.New(store, tgToken, tgApiToken)
	dbCtrl := dbController.NewDBController(store, accessToken)

	servers := []ServerInterface{ordReceiver, tgWorker, dbCtrl}

	errCh := make(chan error, len(servers))

	starter(servers, errCh, &wg, ctx)

	// errors handle
	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			log.Println(err)
		}
	}

	ctxT, canc := context.WithTimeout(context.Background(), time.Second*15)
	defer canc()

	tgWorker.Shutdown(ctxT)

	stop()

	wg.Wait()

	close(errCh)

	for err := range errCh {
		log.Println(err)
	}
}

func initAllStaticVars() {
	tgToken = strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	pathToSQLite = strings.TrimSpace(os.Getenv("SQLITE_PATH"))
	//domain = strings.TrimSpace(os.Getenv("DOMAIN"))
	tgApiToken = strings.TrimSpace(os.Getenv("TELEGRAM_API_TOKEN"))
	accessToken = strings.TrimSpace(os.Getenv("ACCESS_TOKEN"))
	kafkaInstance = strings.TrimSpace(os.Getenv("KAFKA_INSTANCE"))

	if slices.Contains([]string{tgToken, pathToSQLite, tgApiToken, accessToken, kafkaInstance}, "") {
		log.Fatal("Some environment variable is not specified")
	}
}

func prepareDB(path string) *repository.Store {
	db, err := repository.OpenSQLite(context.Background(), repository.OpenOptions{
		Path: path,
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

	err = repository.Migrate(context.Background(), db)
	if err != nil {
		log.Fatal("Cannot start the application because migration of SQLite failed:", err)
	}

	return repository.NewStore(db)
}

func starter(servers []ServerInterface, eCh chan<- error, wg *sync.WaitGroup, ctx context.Context) {
	for _, server := range servers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := server.ListenAndServe(ctx)
			eCh <- err
		}()
	}
}
