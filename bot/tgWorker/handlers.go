package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type TGSendRequest struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

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
	tk := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")

	if tk != TGToken {
		return
	}

	var entry UpdatesRequest

	err := tgWrkr.readJSON(w, r, &entry)
	if err != nil {
		//_ = tgWrkr.errorJSON(w, err)
		log.Println("readJSON() method in tgUpdates() method cannot decode the request:", err)
		return
	}

	if !rr.checkLastAppeal(entry.Message.Chat.ID) {
		log.Printf("Too much requests from user by %d ID", entry.Message.Chat.ID)
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

	userID, err := tgWrkr.store.ConsumeToken(context.Background(), token)
	if err != nil {
		log.Println("ConsumeToken() method from tgUpdates() failed:", err)
		return
	}

	chatID := entry.Message.Chat.ID

	err = tgWrkr.store.UpsertBinding(context.Background(), userID, chatID)
	if err != nil {
		log.Println("Cannot UpsertBinding:", err)
	}

	err = tgWrkr.sendMessage(chatID)
	if err != nil {
		log.Println(err)
	}
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

func (tgWrkr *TGWorker) sendMessage(chatID int64) error {
	var payload TGSendRequest

	payload.ChatID = chatID

	payload.Text = "✅Вы успешно подключились к отслеживанию ваших заказов!"

	entry, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", TGToken)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(entry))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if resp == nil {
		return errors.New("Response is not specified")
	}

	defer resp.Body.Close()

	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return errors.New("telegram returned http status code is another then statusOK")
	}

	return nil
}
