package telegram_test

import (
	telegram "Stroy1ClickBot/bot/tgWorker"
	"Stroy1ClickBot/repository"
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func openTestSQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:cleaner_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Minimal schema required by CleanupExpiredTokens.
	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS bind_tokens (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  token_hash BLOB NOT NULL,
  user_id INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  used_at INTEGER NULL
);
`)
	if err != nil {
		_ = db.Close()
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func TestCleaner_CleansExpiredAndUsedTokens_StopsOnContextCancel(t *testing.T) {
	db := openTestSQLite(t)
	defer db.Close()

	st := repository.NewStore(db)
	//ctx := context.Background()

	now := time.Now().UTC().Unix()

	// Insert: expired token
	_, err := db.Exec(`INSERT INTO bind_tokens(token_hash, user_id, expires_at, used_at) VALUES(?, ?, ?, NULL)`, []byte("expired"), 1, now-10)
	if err != nil {
		t.Fatalf("insert expired: %v", err)
	}
	// Insert: used token (not expired but used_at not null)
	_, err = db.Exec(`INSERT INTO bind_tokens(token_hash, user_id, expires_at, used_at) VALUES(?, ?, ?, ?)`, []byte("used"), 1, now+3600, now)
	if err != nil {
		t.Fatalf("insert used: %v", err)
	}
	// Insert: valid token
	_, err = db.Exec(`INSERT INTO bind_tokens(token_hash, user_id, expires_at, used_at) VALUES(?, ?, ?, NULL)`, []byte("valid"), 1, now+3600)
	if err != nil {
		t.Fatalf("insert valid: %v", err)
	}

	cln := telegram.NewCleaner(st, 10*time.Millisecond)

	runCtx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- cln.Clean(runCtx)
	}()

	// Wait until cleanup likely executed (with a deadline to avoid flakiness).
	deadline := time.Now().Add(300 * time.Millisecond)
	for {
		var cnt int
		err = db.QueryRow(`SELECT COUNT(*) FROM bind_tokens`).Scan(&cnt)
		if err != nil {
			t.Fatalf("count tokens: %v", err)
		}
		if cnt == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected 1 token remaining after cleanup; got %d", cnt)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("clean returned error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("cleaner did not stop after context cancel")
	}
}
