package order

import (
	telegram "Stroy1ClickBot/bot/tgWorker"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	kafka "github.com/segmentio/kafka-go"
)

type TGSendRequest struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type Commiter interface {
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
}

func (ordR *OrderReceiver) receiveAndSend(ctx context.Context) error {
	reader := ordR.getReaderFromKafka()
	defer reader.Close()

	for {
		select {
		case <-ctx.Done():
			err := reader.Close()
			return err
		default:
			message, err := reader.FetchMessage(ctx)
			if err != nil {
				log.Println("Cannot fetch message from kafka:", err)
				continue
			}

			go func(com Commiter) {
				err = ordR.sendMessage(message)
				if err != nil {
					return
				}
				err = com.CommitMessages(ctx, message)
				if err != nil {
					log.Printf("Failed to commit the message by key: %s, value: %s; error: %s\n",
						string(message.Key),
						string(message.Value),
						err)
				}
			}(reader)
		}
	}
}

func (ordR *OrderReceiver) getReaderFromKafka() *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{ordR.instance},
		GroupID:  "tg-bot",
		Topic:    "order-created-events",
		MaxBytes: 1e8,
	})
}

func (ordR *OrderReceiver) sendMessage(message kafka.Message) error {
	var entry Order

	dec := json.NewDecoder(bytes.NewReader(message.Value))

	err := dec.Decode(&entry)
	if err != nil {
		log.Println("User sending error: Error decode JSON in the Decode method:", err)
		return err
	}

	chatID, ok, err := ordR.store.GetChatID(context.Background(), entry.UserID)

	if err != nil {
		log.Printf("User sending error: Failed to retrieve the ChatID by %d UserID: %s\n", entry.UserID, err)
		return err
	} else if !ok {
		log.Println(err)
		return err
	}

	resp, err := ordR.sendMessageToUser(chatID, &entry)

	if err != nil {
		log.Println(err)
		return err
	} else if resp.StatusCode != http.StatusOK {
		log.Println("User sending error: Telegram returned another http status code then StatusOK (200):", resp.StatusCode)
		return err
	}

	bindings, err := ordR.store.GetAllBindingsByUserID(context.Background(), entry.UserID)
	if err != nil {
		log.Printf("Admin sending error: Failed to retrieve binding by %d UserID: %s\n", entry.UserID, err)
		return nil
	}

	username := bindings[0].Username

	resp, err = ordR.sendMessageToAdmin(&entry, username)

	if err != nil {
		log.Println("Admin sending error: writeJSON in receiveAndSend() method failed:", err)
		return nil
	} else if resp.StatusCode != http.StatusOK {
		log.Println("Admin sending error: Telegram returned another http status code then StatusOK (200):", resp.StatusCode)
		return nil
	}

	return nil
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

func (ordR *OrderReceiver) formatingUserMessage(ord *Order) string {
	var message string

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

	var legalForm string
	switch ord.LegalForm {
	case LLC:
		legalForm = "ООО"
	case IE:
		legalForm = "ИП"
	}

	var builder strings.Builder

	for i, item := range ord.OrderItems {
		var unit string

		switch item.Unit {
		case PIECE:
			unit = "Штук"
		case KG:
			unit = "Килограмм"
		case GRAM:
			unit = "Грамм"
		case LITER:
			unit = "Литров"
		case ML:
			unit = "Миллилитров"
		case METER:
			unit = "Метров"
		case PACK:
			unit = "Упаковок"
		}

		_, _ = fmt.Fprintf(&builder, "\t%d:\n\t\t\tID Продукта: %d\n"+
			"\t\t\tКоличество:%d %s\n\n", i, item.ProductID, item.Quantity, unit)
	}

	items := builder.String()

	message = fmt.Sprintf("ℹ️Информация по заказу пользователя:\n\n"+
		"🆔ID заказа: %d\n"+
		"%sСтатус заказа: %s\n"+
		"🪪ID пользователя: %d\n"+
		"Контактное имя: %s\n"+
		"Email: %s\n"+
		"📱Номер телефона пользователя: %s\n"+
		"Название организации: %s\n"+
		"Форма организации: %s\n"+
		"ИНН: %s\n"+
		"КПП: %s\n"+
		"Адрес доставки: %s\n"+
		"📝Запись:%s\n"+
		"🕐Создан: %s\n"+
		"🕝Обновлён: %s\n"+
		"🛒Товары:\n\n%s",
		ord.ID,
		stateEmj,
		state,
		ord.UserID,
		ord.ContactName,
		ord.ContactEmail,
		ord.ContactPhone,
		ord.LegalName,
		legalForm,
		ord.INN,
		ord.KPP,
		ord.DeliveryAddress,
		ord.Notes,
		ord.CreatedAt.String(),
		ord.UpdatedAt.String(),
		items)

	return message
}

func (ordR *OrderReceiver) sendMessageToAdmin(ord *Order, username string) (*http.Response, error) {
	var payload TGSendRequest

	adminChatID, err := ordR.store.GetAdminChatID(context.Background())
	if err != nil {
		return nil, err
	}

	payload.ChatID = adminChatID
	payload.Text = ordR.formatingAdminMessage(ord, username)

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

func (ordR *OrderReceiver) formatingAdminMessage(ord *Order, username string) string {
	var message string

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

	var legalForm string
	switch ord.LegalForm {
	case LLC:
		legalForm = "ООО"
	case IE:
		legalForm = "ИП"
	}

	var builder strings.Builder

	for i, item := range ord.OrderItems {
		var unit string

		switch item.Unit {
		case PIECE:
			unit = "Штук"
		case KG:
			unit = "Килограмм"
		case GRAM:
			unit = "Грамм"
		case LITER:
			unit = "Литров"
		case ML:
			unit = "Миллилитров"
		case METER:
			unit = "Метров"
		case PACK:
			unit = "Упаковок"
		}

		_, _ = fmt.Fprintf(&builder, "\t%d:\n\t\t\tID Продукта: %d\n"+
			"\t\t\tКоличество:%d %s\n\n", i, item.ProductID, item.Quantity, unit)
	}

	items := builder.String()

	message = fmt.Sprintf("ℹ️Информация по заказу пользователя:\n\n"+
		"🆔ID заказа: %d\n"+
		"%sСтатус заказа: %s\n"+
		"🪪ID пользователя: %d\n"+
		"Контактное имя: %s\n"+
		"Email: %s\n"+
		"📱Номер телефона пользователя: %s\n"+
		"Название организации: %s\n"+
		"Форма организации: %s\n"+
		"ИНН: %s\n"+
		"КПП: %s\n"+
		"Адрес доставки: %s\n"+
		"👤Username: @%s\n"+
		"📝Запись:%s\n"+
		"🕐Создан: %s\n"+
		"🕝Обновлён: %s\n"+
		"🛒Товары:\n\n%s",
		ord.ID,
		stateEmj,
		state,
		ord.UserID,
		ord.ContactName,
		ord.ContactEmail,
		ord.ContactPhone,
		ord.LegalName,
		legalForm,
		ord.INN,
		ord.KPP,
		ord.DeliveryAddress,
		username,
		ord.Notes,
		ord.CreatedAt.String(),
		ord.UpdatedAt.String(),
		items)

	return message
}
