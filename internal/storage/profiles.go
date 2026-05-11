package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type ModProfile struct {
	ID        int64  `db:"id"`
	GameID    int64  `db:"game_id"`
	Name      string `db:"name"`
	IsActive  bool   `db:"is_active"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
}

func (s *Store) CreateProfile(ctx context.Context, gameID int64, name string) (profile ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("insert profile row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return ModProfile{}, errors.New("store is not open")
	}

	name, err = normalizeProfileName(name)
	if err != nil {
		return ModProfile{}, err
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO profiles (game_id, name)
		VALUES (?, ?)
	`, gameID, name)
	if err != nil {
		return ModProfile{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return ModProfile{}, fmt.Errorf("get created profile id: %w", err)
	}

	return getProfileByID(ctx, s.db, id)
}

func (s *Store) ListProfiles(ctx context.Context, gameID int64) (profiles []ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select profiles: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}

	err = s.db.SelectContext(ctx, &profiles, `
		SELECT id, game_id, name, is_active, created_at, updated_at
		FROM profiles
		WHERE game_id = ?
		ORDER BY is_active DESC, LOWER(name), id
	`, gameID)
	if err != nil {
		return nil, err
	}

	return profiles, nil
}

func (s *Store) RenameProfile(ctx context.Context, profileID int64, name string) (profile ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update profile name: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return ModProfile{}, errors.New("store is not open")
	}

	name, err = normalizeProfileName(name)
	if err != nil {
		return ModProfile{}, err
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE profiles
		SET name = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, profileID)
	if err != nil {
		return ModProfile{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return ModProfile{}, fmt.Errorf("get renamed profile count: %w", err)
	}
	if affected == 0 {
		return ModProfile{}, sql.ErrNoRows
	}

	return getProfileByID(ctx, s.db, profileID)
}

func (s *Store) DeleteProfile(ctx context.Context, profileID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("delete profile row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}

	_, err = s.db.ExecContext(ctx, `
		DELETE FROM profiles
		WHERE id = ?
	`, profileID)
	return err
}

func (s *Store) ActivateProfile(ctx context.Context, gameID int64, profileID int64) (profile ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("set active profile row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return ModProfile{}, errors.New("store is not open")
	}

	err = withTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		profile, err = getProfileByID(ctx, tx, profileID)
		if err != nil {
			return err
		}
		if profile.GameID != gameID {
			return fmt.Errorf("profile %d does not belong to game %d", profileID, gameID)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE profiles
			SET is_active = 0,
				updated_at = CURRENT_TIMESTAMP
			WHERE game_id = ?
				AND is_active = 1
		`, gameID); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE profiles
			SET is_active = 1,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, profileID); err != nil {
			return err
		}

		profile, err = getProfileByID(ctx, tx, profileID)
		return err
	})
	if err != nil {
		return ModProfile{}, err
	}

	return profile, nil
}

func (s *Store) ClearActiveProfile(ctx context.Context, gameID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("clear active profile row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE profiles
		SET is_active = 0,
			updated_at = CURRENT_TIMESTAMP
		WHERE game_id = ?
			AND is_active = 1
	`, gameID)
	return err
}

func (s *Store) GetActiveProfile(ctx context.Context, gameID int64) (profile ModProfile, found bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select active profile: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return ModProfile{}, false, errors.New("store is not open")
	}

	err = s.db.GetContext(ctx, &profile, `
		SELECT id, game_id, name, is_active, created_at, updated_at
		FROM profiles
		WHERE game_id = ?
			AND is_active = 1
	`, gameID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ModProfile{}, false, nil
		}

		return ModProfile{}, false, err
	}

	return profile, true, nil
}

type profileGetter interface {
	GetContext(context.Context, any, string, ...any) error
}

func getProfileByID(ctx context.Context, db profileGetter, profileID int64) (ModProfile, error) {
	var profile ModProfile
	err := db.GetContext(ctx, &profile, `
		SELECT id, game_id, name, is_active, created_at, updated_at
		FROM profiles
		WHERE id = ?
	`, profileID)
	if err != nil {
		return ModProfile{}, err
	}

	return profile, nil
}

func normalizeProfileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("profile name is required")
	}

	return name, nil
}
