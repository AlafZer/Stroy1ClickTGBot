package telegram

import (
	"Stroy1ClickBot/repository"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestRequestResistor_CheckLastAppeal(t *testing.T) {
	r := &RequestResistor{lastAppeal: make(map[int64]time.Time)}

	chatID := int64(42)
	if ok := r.checkLastAppeal(chatID); !ok {
		t.Fatal("first appeal should pass")
	}
	if ok := r.checkLastAppeal(chatID); ok {
		t.Fatal("second immediate appeal should be rejected")
	}

	// Simulate old appeal time without sleeping.
	r.lastAppeal[chatID] = time.Now().Add(-minTimeDifferent - 50*time.Millisecond)
	if ok := r.checkLastAppeal(chatID); !ok {
		t.Fatal("appeal after minTimeDifferent should pass")
	}
}

func TestNew_SetsGlobalsAndInitializesResistor(t *testing.T) {
	// Minimal store instance.
	db, err := sql.Open("sqlite", "file:tgworker_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	st := repository.NewStore(db)

	w := New(st, "token123", "apiToken456")
	if w == nil {
		t.Fatal("worker should not be nil")
	}
	if TGToken != "token123" {
		t.Fatalf("TGToken not set: %q", TGToken)
	}
	if TokenAPI != "apiToken456" {
		t.Fatalf("TokenAPI not set: %q", TokenAPI)
	}
	if rr == nil {
		t.Fatal("request resistor should be initialized")
	}
	if rr.lastAppeal == nil {
		t.Fatal("request resistor map should be initialized")
	}
	if w.server == nil {
		t.Fatal("http server should be initialized")
	}
}
