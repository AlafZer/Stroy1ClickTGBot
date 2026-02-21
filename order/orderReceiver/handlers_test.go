package order

import (
	telegram "Stroy1ClickBot/bot/tgWorker"
	"Stroy1ClickBot/repository"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	kafka "github.com/segmentio/kafka-go"
	_ "modernc.org/sqlite"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFormatingUserMessage_ContainsKeyFields(t *testing.T) {
	ordR := &OrderReceiver{}
	ord := &Order{
		ID:          99,
		UserID:      7,
		OrderStatus: Created,
		Notes:       "note",
		CreatedAt:   time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 2, 18, 11, 0, 0, 0, time.UTC),
		OrderItems:  []OrderItem{{ProductID: 123, Quantity: 2}},
	}

	msg := ordR.formatingUserMessage(ord)
	if !strings.Contains(msg, "ID заказа: 99") {
		t.Fatalf("expected order ID, got: %s", msg)
	}
	if !strings.Contains(msg, "ID пользователя: 7") {
		t.Fatalf("expected user ID, got: %s", msg)
	}
	if !strings.Contains(msg, "Статус заказа: Создан") {
		t.Fatalf("expected status 'Создан', got: %s", msg)
	}
	if !strings.Contains(msg, "ID Продукта: 123") {
		t.Fatalf("expected item product id, got: %s", msg)
	}
}

func TestFormatingAdminMessage_ContainsUsernameAndPhone(t *testing.T) {
	ordR := &OrderReceiver{}
	ord := &Order{
		ID:           1,
		UserID:       2,
		OrderStatus:  Paid,
		ContactPhone: "+357000000",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		OrderItems:   []OrderItem{{ProductID: 10, Quantity: 1}},
	}

	msg := ordR.formatingAdminMessage(ord, "alice")
	if !strings.Contains(msg, "Username: @alice") {
		t.Fatalf("expected username, got: %s", msg)
	}
	if !strings.Contains(msg, "Номер телефона пользователя") {
		t.Fatalf("expected phone label, got: %s", msg)
	}
	if !strings.Contains(msg, "Оплачен") {
		t.Fatalf("expected status 'Оплачен', got: %s", msg)
	}
}

func TestSendMessage_SendsTelegramRequestForLinkedUser(t *testing.T) {
	// Use dummy token.
	telegram.TGToken = "dummy"

	// In-memory DB with minimal schema for GetChatID.
	db, err := sql.Open("sqlite", "file:order_send_message?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`CREATE TABLE tg_bindings (user_id INTEGER PRIMARY KEY, chat_id INTEGER NOT NULL);`)
	if err != nil {
		t.Fatalf("create tg_bindings: %v", err)
	}
	_, err = db.Exec(`INSERT INTO tg_bindings(user_id, chat_id) VALUES(?, ?)`, 7, 777)
	if err != nil {
		t.Fatalf("insert binding: %v", err)
	}

	st := repository.NewStore(db)
	ordR := &OrderReceiver{store: st}

	// Intercept outbound HTTP.
	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()

	var got []TGSendRequest
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.Contains(r.URL.String(), "/botdummy/sendMessage") {
			t.Fatalf("unexpected url: %s", r.URL.String())
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		var req TGSendRequest
		if err := json.Unmarshal(b, &req); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		got = append(got, req)

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString("ok")),
			Request:    r,
		}, nil
	})

	ord := Order{
		ID:          1,
		UserID:      7,
		OrderStatus: Created,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		OrderItems:  []OrderItem{{ProductID: 1, Quantity: 1}},
	}
	payload, _ := json.Marshal(ord)
	msg := kafka.Message{Value: payload}

	if err := ordR.sendMessage(msg); err != nil {
		t.Fatalf("sendMessage returned error: %v", err)
	}

	if len(got) < 1 {
		t.Fatalf("expected at least 1 telegram call, got %d", len(got))
	}
	if got[0].ChatID != 777 {
		t.Fatalf("expected chat_id=777, got %d", got[0].ChatID)
	}
	if got[0].Text == "" {
		t.Fatal("expected non-empty text")
	}
}

func TestSendMessage_NoBinding_DoesNotCallTelegram(t *testing.T) {
	telegram.TGToken = "dummy"

	db, err := sql.Open("sqlite", "file:order_send_message_nobind?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`CREATE TABLE tg_bindings (user_id INTEGER PRIMARY KEY, chat_id INTEGER NOT NULL);`)
	if err != nil {
		t.Fatalf("create tg_bindings: %v", err)
	}

	st := repository.NewStore(db)
	ordR := &OrderReceiver{store: st}

	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()

	calls := 0
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("ok")),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})

	ord := Order{ID: 1, UserID: 7, OrderStatus: Created, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	payload, _ := json.Marshal(ord)

	_ = ordR.sendMessage(kafka.Message{Value: payload})

	if calls != 0 {
		t.Fatalf("expected 0 http calls, got %d", calls)
	}
	_ = context.Background()
}
