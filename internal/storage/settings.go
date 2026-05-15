package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const GlobalModStorageRootSettingKey = "mods.global_storage_root"

func (s *Store) GetSetting(ctx context.Context, key string) (value string, found bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get setting %q: %w", key, err)
		}
	}()

	if s == nil || s.db == nil {
		return "", false, errors.New("store is not open")
	}

	err = s.db.GetContext(ctx, &value, `
		SELECT value
		FROM settings
		WHERE key = ?
	`, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}

		return "", false, err
	}

	return value, true, nil
}

func (s *Store) GetGlobalModStorageRoot(ctx context.Context) (root string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("read global mod storage root setting: %w", err)
		}
	}()

	value, found, err := s.GetSetting(ctx, GlobalModStorageRootSettingKey)
	if err != nil {
		return "", err
	}
	if !found {
		return "", nil
	}

	return cleanOptionalPath(value), nil
}

func (s *Store) SetGlobalModStorageRoot(ctx context.Context, path string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("write global mod storage root setting: %w", err)
		}
	}()

	if err := s.SetSetting(ctx, GlobalModStorageRootSettingKey, cleanOptionalPath(path)); err != nil {
		return err
	}

	return s.RefreshGameModStoragePaths(ctx)
}

func (s *Store) SetSetting(ctx context.Context, key string, value string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("set setting %q: %w", key, err)
		}
	}()

	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = CURRENT_TIMESTAMP
	`, key, value)
	if err != nil {
		return err
	}

	return nil
}
