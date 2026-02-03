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

// receiveAndSend find user_id -> chat_id match, formatting user and admin messages
// then it'll send them to recipient'
// receiveAndSend godoc
// @Summary      Send order notification to Telegram user
// @Description  Finds chat_id by userId and sends formatted order message to user.
// @Tags         order
// @Accept       json
// @Produce      json
// @Param        order body Order true "Order DTO"
// @Success      200 {object} JSONResponse
// @Failure      400 {object} JSONResponse
// @Failure      404 {object} JSONResponse
// @Failure      500 {object} JSONResponse
// @Router       /api/v1/telegram/send [post]
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

	resp, err := ordR.sendMessageToUser(chatID, &entry)

	if err != nil {
		_ = ordR.errorJSON(w, err, http.StatusInternalServerError)
		log.Println(err)
		return
	} else if resp.StatusCode != http.StatusOK {
		_ = ordR.errorJSON(w, errors.New("telegram returned another http status code then StatusOK (200)"), resp.StatusCode)
		log.Println("Telegram returned another http status code then StatusOK (200):", resp.StatusCode)
		return
	}

	var response JSONResponse

	response.Error = false
	response.Message = "Order was successfully send"

	err = ordR.writeJSON(w, http.StatusOK, response)
	if err != nil {
		log.Println("writeJSON in receiveAndSend() method failed:", err)
	}
}

func (ordR *OrderReceiver) formatingUserMessage(ord *Order) string {
	var message string

	items := ""
	var state string
	var stateEmj string

	switch ord.OrderStatus {
	case Created:
		state = "Создан"
		stateEmj = "✅"
	case Paid:
		state = "Оплачен"
		stateEmj = "💳"
	case Shipped:
		state = "Отправлен"
		stateEmj = "🛫"
	case Delivered:
		state = "Доставлен"
		stateEmj = "🛬"
	case Canceled:
		state = "Отменён"
		stateEmj = "❌"
	}

	for i, item := range ord.OrderItems {
		items += fmt.Sprintf("\t%d:\n\t\t\tID Продукта: %d\n\t\t\tКоличество:%d\n\n", i, item.ProductID, item.Quantity)
	}

	message = fmt.Sprintf("ℹ️Информация по вашему заказу\n\n🆔ID заказа: %d\n%sСтатус заказа: %s\n🪪ID пользователя: %d\n📝Запись:%s\n🕐Создан: %s\n🕝Обновлён: %s\n🛒Товары:\n\n%s",
		ord.ID, stateEmj, state, ord.UserID, ord.Notes, ord.CreatedAt.String(), ord.UpdatedAt.String(), items)

	return message
}

func (ordR *OrderReceiver) sendMessageToUser(chatID int64, ord *Order) (*http.Response, error) {
	var payload TGSendRequest

	payload.ChatID = chatID
	payload.Text = ordR.formatingUserMessage(ord)

	message, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegram.TGToken)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(message))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)

	return resp, err
}

//func (ordR *OrderReceiver) formatingAdminMessage(ord *Order) string {
//	var message string
//
//	items := ""
//	var state string
//	var stateEmj string
//
//	switch ord.OrderStatus {
//	case Created:
//		state = "Создан"
//		stateEmj = "✅"
//	case Paid:
//		state = "Оплачен"
//		stateEmj = "💳"
//	case Shipped:
//		state = "Отправлен"
//		stateEmj = "🛫"
//	case Delivered:
//		state = "Доставлен"
//		stateEmj = "🛬"
//	case Canceled:
//		state = "Отменён"
//		stateEmj = "❌"
//	}
//
//	for i, item := range ord.OrderItems {
//		items += fmt.Sprintf("\t%d:\n\t\t\tID Продукта: %d\n\t\t\tКоличество:%d\n\n", i, item.ProductID, item.Quantity)
//	}
//
//	message = fmt.Sprintf("ℹ️Информация по вашему заказу\n\n🆔ID заказа: %d\n%sСтатус заказа: %s\n🪪ID пользователя: %d\n📝Запись:%s\n🕐Создан: %s\n🕝Обновлён: %s\n🛒Товары:\n\n%s",
//		ord.ID, stateEmj, state, ord.UserID, ord.Notes, ord.CreatedAt.String(), ord.UpdatedAt.String(), items)
//
//	return message
//}
