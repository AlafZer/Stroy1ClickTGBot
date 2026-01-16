package order

import (
	"Stroy1ClickBot/storage"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	webPort = "8080"
)

type OrderReceiver struct {
	server *http.Server
	store  *storage.Store
}

func New() *OrderReceiver {
	ordR := &OrderReceiver{}

	srv := &http.Server{
		Addr:    ":" + webPort,
		Handler: ordR.routes(),
	}

	ordR.server = srv

	return ordR
}

func (ordR *OrderReceiver) Listen() error {
	log.Println("Starting OrderReceiverServer on port:", webPort)
	err := ordR.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (ordR *OrderReceiver) Shutdown() {
	ctxT, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	// Graceful Shutdown
	if err := ordR.server.Shutdown(ctxT); err != nil {
		log.Println("Failed to Shutdown server:", err)
	}
}

func (ordR *OrderReceiver) formatingMessage(ord *Order) string {
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
		items += fmt.Sprintf("\t%d:\n\t🆔ID Продукта: %d\n\t💵Стоимость:v%d\n\n", i, item.ProductID, item.Quantity)
	}

	message = fmt.Sprintf("ℹ️Информация по вашему заказу\n\n🆔ID заказа: %d\n%sСтатус заказа: %s\n🪪ID пользователя: %d\n📝Запись:%s\n🕐Создан: %T\n🕝Обновлён: %T\n🧺Товары:\n\n%s",
		ord.ID, stateEmj, state, ord.UserID, ord.Notes, ord.CreatedAt, ord.UpdatedAt, items)

	return message
}
