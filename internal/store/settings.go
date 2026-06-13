package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (db *DB) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := db.Pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&value)
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("setting %q not found", key)
	}
	return value, err
}

func (db *DB) SetSetting(ctx context.Context, key, value string) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO settings (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
		key, value)
	return err
}
