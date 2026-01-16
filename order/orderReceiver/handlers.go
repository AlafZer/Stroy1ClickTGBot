package order

import (
	telegram "Stroy1ClickBot/bot/tgWorker"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

type TGSendRequest struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

func (ordR *OrderReceiver) receiveAndSend(w http.ResponseWriter, r *http.Request) {
	var entry Order

	err := readJSON(w, r, &entry)
	if err != nil {
		log.Println("Error reading JSON in the receive method:", err)

		return
	}

	chatID, ok, err := ordR.store.GetChatID(context.Background(), entry.UserID)

	if err != nil {
		_ = ordR.errorJSON(w, err)
		log.Println(err)
		return
	} else if !ok {
		_ = ordR.errorJSON(w, errors.New("user not linked to telegram"))
		log.Println(err)
		return
	}

	var payload TGSendRequest

	payload.ChatID = chatID
	payload.Text = ordR.formatingMessage(&entry)

	message, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegram.TGToken)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(message))
	if err != nil {
		ordR.errorJSON(w, err)
		log.Println(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}

	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		ordR.errorJSON(w, err, http.StatusInternalServerError)
		log.Println(err)
		return
	} else if resp.StatusCode != http.StatusOK {
		ordR.errorJSON(w, errors.New("Telegram returned another http status code then StatusOK (200)"), resp.StatusCode)
		log.Println("Telegram returned another http status code then StatusOK (200):", resp.StatusCode)
		return
	}
}
