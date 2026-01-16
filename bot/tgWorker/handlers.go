package telegram

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type LinkRequest struct {
	UserID int64 `json:"user_id"`
}

type LinkResponse struct {
	LinkURL string `json:"link_url"`
}

func (tgWrkr *TGWorker) tgLink(w http.ResponseWriter, r *http.Request) {
	var entry LinkRequest

	err := tgWrkr.readJSON(w, r, &entry)
	if err != nil {
		_ = tgWrkr.errorJSON(w, err)
		log.Println(err)
		return
	}

	token, _, err := tgWrkr.store.CreateToken(context.Background(), entry.UserID, 20*time.Minute)
	if err != nil {
		_ = tgWrkr.errorJSON(w, err)
		log.Println(err)
		return
	}

	url := tgWrkr.URL(token)

	var response LinkResponse

	response.LinkURL = url

	err = tgWrkr.writeJSON(w, http.StatusOK, response)

	if err != nil {
		log.Println("writeJSON in tgLink() method failed:", err)
	}
}

func (tgWrkr *TGWorker) tgUpdates(w http.ResponseWriter, r *http.Request) {
	var entry UpdatesRequest

	err := tgWrkr.readJSON(w, r, &entry)
	if err != nil {
		//_ = tgWrkr.errorJSON(w, err)
		log.Println("readJSON() method in tgUpdates() method cannot decode the request:", err)
		return
	}

	token, ok, err := tgWrkr.extractToken(&entry)
	if err != nil {
		//_ = tgWrkr.errorJSON(w, err)
		log.Println("extractToken from update query failed:", err)
		return
	} else if !ok {
		//_ = tgWrkr.errorJSON(w, errors.New("Invalid token"))
		log.Println("extractToken from update query failed:", err)
		return
	}

	userId, err := tgWrkr.store.ConsumeToken(context.Background(), token)
	if err != nil {
		log.Println("ConsumeToken() method from tgUpdates() failed:", err)
		return
	}

	chatId := entry.Message.Chat.ID

	err = tgWrkr.store.UpsertBinding(context.Background(), userId, chatId)
}

func (tgWrkr *TGWorker) URL(token string) string {
	return fmt.Sprintf("https://t.me/Stroy1ClickOrderBot?start=%s", token)
}

func (tgWrkr *TGWorker) extractToken(update *UpdatesRequest) (string, bool, error) {
	query := update.Message.Text

	if strings.HasPrefix(query, "/start ") {
		token, _ := strings.CutPrefix(query, "/start ")
		if token == "" || len(token) > 64 {
			return "", false, nil
		}
		return token, true, nil
	}

	return "", false, errors.New("Invalid query")
}
