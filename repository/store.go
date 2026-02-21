package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

type BindTokens struct {
	ID        int64     `json:"id"`
	TokenHash string    `json:"token_hash"`
	UserID    int64     `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	UsedAt    int       `json:"used_at"`
}

type TGBindings struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	ChatID    int64     `json:"chat_id"`
	BoundAt   time.Time `json:"bound_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsAdmin   int       `json:"is_admin"`
	Username  string    `json:"username"`
}

var (
	// ErrTokenInvalidOrExpired Возвращаем это при: токен не найден / истёк / уже использован / гонка
	ErrTokenInvalidOrExpired = errors.New("token invalid/expired/used")

	// ErrNotLinked Если привязки нет
	ErrNotLinked = errors.New("user not linked to telegram")
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// CreateToken создаёт новый токен на ttl, инвалидируя старые активные токены пользователя.
// Возвращает сам token (его ты вставляешь в deep-link).
func (s *Store) CreateToken(ctx context.Context, userID int64, ttl time.Duration) (token string, expiresAt time.Time, err error) {
	if userID <= 0 {
		return "", time.Time{}, fmt.Errorf("storage: invalid userID")
	}

	now := time.Now().UTC()
	exp := now.Add(ttl)

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return "", time.Time{}, err
	}
	defer func() { _ = tx.Rollback() }()

	// Инвалидируем все старые активные токены этого userId
	_, err = tx.ExecContext(ctx,
		`UPDATE bind_tokens SET used_at = ?
		 WHERE user_id = ? AND used_at IS NULL;`,
		now.Unix(), userID,
	)
	if err != nil {
		return "", time.Time{}, err
	}

	// Генерим токен: 32 байта -> base64url без "="
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)

	hash := sha256.Sum256([]byte(token))

	_, err = tx.ExecContext(ctx,
		`INSERT INTO bind_tokens(token_hash, user_id, expires_at, used_at)
		 VALUES(?, ?, ?, NULL);`,
		hash[:], userID, exp.Unix(),
	)
	if err != nil {
		return "", time.Time{}, err
	}

	if err := tx.Commit(); err != nil {
		return "", time.Time{}, err
	}

	return token, exp, nil
}

// ConsumeToken атомарно “съедает” токен: проверка (не использован, не истёк) + used_at.
// Возвращает userId, который надо привязать к chatId.
func (s *Store) ConsumeToken(ctx context.Context, token string) (int64, error) {
	if token == "" {
		return 0, ErrTokenInvalidOrExpired
	}

	now := time.Now().UTC().Unix()
	hash := sha256.Sum256([]byte(token))

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	var userID int64
	err = tx.QueryRowContext(ctx,
		`SELECT user_id
		 FROM bind_tokens
		 WHERE token_hash = ?
		   AND used_at IS NULL
		   AND expires_at > ?;`,
		hash[:], now,
	).Scan(&userID)

	if err == sql.ErrNoRows {
		return 0, ErrTokenInvalidOrExpired
	}
	if err != nil {
		return 0, err
	}

	// Помечаем used_at (защита от гонок: условие used_at IS NULL)
	res, err := tx.ExecContext(ctx,
		`UPDATE bind_tokens
		 SET used_at = ?
		 WHERE token_hash = ? AND used_at IS NULL;`,
		now, hash[:],
	)
	if err != nil {
		return 0, err
	}
	aff, _ := res.RowsAffected()
	if aff != 1 {
		return 0, ErrTokenInvalidOrExpired
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return userID, nil
}

// UpsertBinding создаёт/обновляет привязку userId -> chatId.
func (s *Store) UpsertBinding(ctx context.Context, userID, chatID int64, username string) error {
	if userID <= 0 || chatID == 0 {
		return fmt.Errorf("storage: invalid userID/chatID")
	}

	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tg_bindings(user_id, chat_id, tg_username, bound_at, updated_at)
		 VALUES(?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   chat_id = excluded.chat_id,
		   updated_at = excluded.updated_at;`,
		userID, chatID, username, now, now,
	)
	return err
}

// GetChatID возвращает chatId для отправки уведомлений.
func (s *Store) GetChatID(ctx context.Context, userID int64) (chatID int64, ok bool, err error) {
	if userID <= 0 {
		return 0, false, fmt.Errorf("storage: invalid userID")
	}

	err = s.db.QueryRowContext(ctx,
		`SELECT chat_id FROM tg_bindings WHERE user_id = ?;`,
		userID,
	).Scan(&chatID)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return chatID, true, nil
}

// DeleteBinding (опционально) — “отключить Telegram”.
func (s *Store) DeleteBinding(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("storage: invalid userID")
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM tg_bindings WHERE user_id = ?;`, userID)
	return err
}

// CleanupExpiredTokens (опционально) — чистка токенов (например, раз в час).
func (s *Store) CleanupExpiredTokens(ctx context.Context) (deleted int64, err error) {
	now := time.Now().UTC().Unix()
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM bind_tokens
		 WHERE (expires_at <= ?) OR (used_at IS NOT NULL);`,
		now,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// GetAdminChatID returns one admin chatID
func (s *Store) GetAdminChatID(ctx context.Context) (int64, error) {
	var admin int64

	query := `SELECT chat_id FROM tg_bindings WHERE is_admin = 1`

	err := s.db.QueryRowContext(ctx, query).Scan(&admin)
	if err != nil {
		return -1, err
	}

	return admin, nil
}

// SetAdminRole changes field is_admin to current state in the database by userId
func (s *Store) SetAdminRole(ctx context.Context, userId int64, state bool) error {
	query := `UPDATE tg_bindings SET is_admin = ? WHERE user_id = ?`

	_, err := s.db.ExecContext(ctx, query, state, userId)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) GetAllBindings(ctx context.Context) ([]*TGBindings, error) {
	var bindings []*TGBindings

	query := `SELECT * FROM tg_bindings`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var binding TGBindings
		err = rows.Scan(&binding.ID, &binding.UserID, &binding.ChatID, &binding.BoundAt, &binding.UpdatedAt, &binding.IsAdmin)
		if err != nil {
			return bindings, err
		}

		bindings = append(bindings, &binding)
	}

	return bindings, nil
}

func (s *Store) GetAllBindingsByUserID(ctx context.Context, userID int64) ([]*TGBindings, error) {
	var bindings []*TGBindings

	query := `SELECT * FROM tg_bindings WHERE user_id = ?`
	row := s.db.QueryRowContext(ctx, query, userID)

	var binding TGBindings
	err := row.Scan(&binding.ID, &binding.UserID, &binding.ChatID, &binding.BoundAt, &binding.UpdatedAt, &binding.IsAdmin)
	if err != nil {
		return nil, err
	}

	bindings = append(bindings, &binding)

	return bindings, nil
}

func (s *Store) GetAllTokens(ctx context.Context) ([]*BindTokens, error) {
	var tokens []*BindTokens

	query := `SELECT * FROM bind_tokens`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var token BindTokens
		err = rows.Scan(&token.ID, &token.TokenHash, &token.UserID, &token.ExpiresAt, &token.UsedAt)
		if err != nil {
			return tokens, err
		}

		tokens = append(tokens, &token)
	}

	return tokens, nil
}
