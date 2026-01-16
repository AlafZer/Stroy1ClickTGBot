package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type OpenOptions struct {
	Path string // например "./data/notification.db"
}

func OpenSQLite(ctx context.Context, opt OpenOptions) (*sql.DB, error) {
	if opt.Path == "" {
		return nil, fmt.Errorf("storage: empty db path")
	}

	// Создаём директорию под файл БД (если нужно)
	dir := filepath.Dir(opt.Path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("storage: mkdir %s: %w", dir, err)
		}
	}

	// modernc драйвер: имя "sqlite"
	// "file:" URI работает нормально. Можно также просто opt.Path, но file: более однозначно.
	dsn := "file:" + opt.Path

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("storage: open sqlite: %w", err)
	}

	// Для SQLite часто ставят 1 соединение, чтобы снизить вероятность SQLITE_BUSY
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	// Пингуем с таймаутом
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("storage: ping: %w", err)
	}

	// PRAGMA (можно менять под себя)
	pragmas := []string{
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA synchronous = NORMAL;`,
		`PRAGMA busy_timeout = 5000;`,
	}
	for _, q := range pragmas {
		if _, err := db.ExecContext(ctx, q); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("storage: pragma failed (%s): %w", q, err)
		}
	}

	return db, nil
}
