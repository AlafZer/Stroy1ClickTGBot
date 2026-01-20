package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Migrate(ctx context.Context, db *sql.DB) error {
	// BOOTSTRAP: таблица должна существовать ДО любых проверок isMigrationApplied
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at INTEGER NOT NULL
		);
	`); err != nil {
		return fmt.Errorf("storage: create schema_migrations: %w", err)
	}

	files, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("storage: list migrations: %w", err)
	}
	sort.Strings(files)

	for _, path := range files {
		version := strings.TrimSuffix(filepath.Base(path), ".sql")

		applied, err := isMigrationApplied(ctx, db, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		body, err := migrationsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("storage: read migration %s: %w", path, err)
		}

		tx, err := db.BeginTx(ctx, &sql.TxOptions{})
		if err != nil {
			return fmt.Errorf("storage: begin tx: %w", err)
		}

		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("storage: apply migration %s: %w", version, err)
		}

		now := time.Now().UTC().Unix()
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?);`,
			version, now,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("storage: record migration %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("storage: commit migration %s: %w", version, err)
		}
	}

	return nil
}

func isMigrationApplied(ctx context.Context, db *sql.DB, version string) (bool, error) {
	var v string
	err := db.QueryRowContext(ctx,
		`SELECT version FROM schema_migrations WHERE version = ?;`,
		version,
	).Scan(&v)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("storage: check migration %s: %w", version, err)
	}
	return true, nil
}
