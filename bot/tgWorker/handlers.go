package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
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

// tgLink returns a link with token which will opened by user
// tgLink godoc
// @Summary      Create Telegram deep-link for binding
// @Description  Generates one-time token and returns t.me link with /start <token>.
// @Tags         telegram
// @Accept       json
// @Produce      json
// @Param        request body LinkRequest true "User id to link"
// @Success      200 {object} LinkResponse
// @Failure      400 {object} JSONResponse
// @Failure      500 {object} JSONResponse
// @Router       /api/v1/telegram/link [post]
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

// tgUpdates receives the update requests from telegramAPI to link the chatID and userID
// tgUpdates godoc
// @Summary      Telegram webhook endpoint
// @Description  Receives updates from Telegram and binds userId to chatId.
// @Tags         telegram
// @Accept       json
// @Produce      json
// @Param        X-Telegram-Bot-Api-Secret-Token header string true "Secret token"
// @Param        update body UpdatesRequest true "Telegram update payload (subset)"
// @Success      200 {string} string "OK"
// @Failure      400 {object} JSONResponse
// @Failure      403 {object} JSONResponse
// @Failure      429 {object} JSONResponse
// @Failure      500 {object} JSONResponse
// @Router       /api/v1/telegram/updates [post]
func (tgWrkr *TGWorker) tgUpdates(w http.ResponseWriter, r *http.Request) {
	tk := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")

	if tk != TokenAPI {
		_ = tgWrkr.errorJSON(w, errors.New("invalid token"), http.StatusForbidden)
		return
	}

	var entry UpdatesRequest

	err := tgWrkr.readJSON(w, r, &entry)
	if err != nil {
		_ = tgWrkr.errorJSON(w, err)
		log.Println("readJSON() method in tgUpdates() method cannot decode the request:", err)
		return
	}

	if !rr.checkLastAppeal(entry.Message.Chat.ID) {
		log.Printf("Too much requests from user by %d ID", entry.Message.Chat.ID)
		_ = tgWrkr.errorJSON(w, errors.New("too much requests"))
		return
	}

	token, ok, err := tgWrkr.extractToken(&entry)

	if err != nil {
		_ = tgWrkr.errorJSON(w, err)
		log.Println("extractToken from update query failed:", err)
		return
	} else if !ok {
		_ = tgWrkr.errorJSON(w, errors.New("invalid token"))
		log.Println("extractToken from update query failed:", err)
		return
	}

	userID, err := tgWrkr.store.ConsumeToken(context.Background(), token)
	if err != nil {
		_ = tgWrkr.errorJSON(w, err)
		log.Println("ConsumeToken() method from tgUpdates() failed:", err)
		return
	}

	chatID := entry.Message.Chat.ID
	username := entry.Message.From.Username

	err = tgWrkr.store.UpsertBinding(context.Background(), userID, chatID, username)
	if err != nil {
		_ = tgWrkr.errorJSON(w, err)
		log.Println("Cannot UpsertBinding:", err)
		return
	}

	err = tgWrkr.sendMessage(chatID)
	if err != nil {
		_ = tgWrkr.errorJSON(w, err)
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

	return "", false, errors.New("invalid query")
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
		return errors.New("response is not specified")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return errors.New("telegram returned http status code is another then statusOK")
	}

	return nil
}

func (tgWrkr *TGWorker) tgDelete(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "UserID"), 10, 64)
	if err != nil {
		log.Printf("Cannot read url parameter UserID in the current url: %s", err)
		_ = tgWrkr.errorJSON(w, err)
		return
	}

	err = tgWrkr.store.DeleteBinding(context.Background(), userID)
	if err != nil {
		log.Printf("Cannot delete the binding with provided userID:%d; error:%s", userID, err)
		_ = tgWrkr.errorJSON(w, err)
		return
	}

	var response JSONResponse

	response.Error = false
	response.Message = "The binding was successfully deleted"

	_ = tgWrkr.writeJSON(w, http.StatusOK, response)
}
