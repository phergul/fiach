package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (s *Store) GetSetting(ctx context.Context, key string) (string, bool, error) {
	if s == nil || s.db == nil {
		return "", false, fmt.Errorf("get setting %q: store is not open", key)
	}

	var value string
	err := s.db.GetContext(ctx, &value, `
		SELECT value
		FROM settings
		WHERE key = ?
	`, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}

		return "", false, fmt.Errorf("get setting %q: %w", key, err)
	}

	return value, true, nil
}

func (s *Store) SetSetting(ctx context.Context, key string, value string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("set setting %q: store is not open", key)
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = CURRENT_TIMESTAMP
	`, key, value)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}

	return nil
}
