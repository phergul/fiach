package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func (s *Store) CreateProfile(ctx context.Context, gameID int64, name string) (profile dbtypes.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("insert profile row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.ModProfile{}, errors.New("store is not open")
	}

	name, err = normalizeProfileName(name)
	if err != nil {
		return dbtypes.ModProfile{}, err
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO profiles (game_id, name)
		VALUES (?, ?)
	`, gameID, name)
	if err != nil {
		return dbtypes.ModProfile{}, mapSQLiteError(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return dbtypes.ModProfile{}, fmt.Errorf("get created profile id: %w", err)
	}

	return getProfileByID(ctx, s.db, id)
}

func (s *Store) DuplicateProfile(ctx context.Context, profileID int64) (profile dbtypes.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("duplicate profile row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.ModProfile{}, errors.New("store is not open")
	}

	err = withTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		originalProfile, err := getProfileByID(ctx, tx, profileID)
		if err != nil {
			return err
		}

		originalProfileMods, err := listProfileMods(ctx, tx, profileID)
		if err != nil {
			return err
		}

		name, err := normalizeProfileName(fmt.Sprintf("%s (copy)", originalProfile.Name))
		if err != nil {
			return err
		}

		result, err := tx.ExecContext(ctx, `
			INSERT INTO profiles (game_id, name)
			VALUES (?, ?)
		`, originalProfile.GameID, name)
		if err != nil {
			return mapSQLiteError(err)
		}

		newProfileID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("get duplicated profile id: %w", err)
		}

		profile, err = getProfileByID(ctx, tx, newProfileID)
		if err != nil {
			return err
		}

		for _, mod := range originalProfileMods {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO profile_mods (profile_id, mod_id, enabled, load_order)
				VALUES (?, ?, ?, ?)
			`, profile.ID, mod.ModID, mod.Enabled, mod.LoadOrder); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return dbtypes.ModProfile{}, err
	}

	return profile, nil
}

func (s *Store) ListProfiles(ctx context.Context, gameID int64) (profiles []dbtypes.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select profiles: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}

	err = s.db.SelectContext(ctx, &profiles, `
		SELECT id, game_id, name, created_at, updated_at
		FROM profiles
		WHERE game_id = ?
		ORDER BY LOWER(name), id
	`, gameID)
	if err != nil {
		return nil, err
	}

	return profiles, nil
}

func (s *Store) RenameProfile(ctx context.Context, profileID int64, name string) (profile dbtypes.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update profile name: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.ModProfile{}, errors.New("store is not open")
	}

	name, err = normalizeProfileName(name)
	if err != nil {
		return dbtypes.ModProfile{}, err
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE profiles
		SET name = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, profileID)
	if err != nil {
		return dbtypes.ModProfile{}, mapSQLiteError(err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return dbtypes.ModProfile{}, fmt.Errorf("get renamed profile count: %w", err)
	}
	if affected == 0 {
		return dbtypes.ModProfile{}, sql.ErrNoRows
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

func (s *Store) GetProfile(ctx context.Context, profileID int64) (profile dbtypes.ModProfile, found bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select profile row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.ModProfile{}, false, errors.New("store is not open")
	}

	profile, err = getProfileByID(ctx, s.db, profileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dbtypes.ModProfile{}, false, nil
		}

		return dbtypes.ModProfile{}, false, err
	}

	return profile, true, nil
}

type profileGetter interface {
	GetContext(context.Context, any, string, ...any) error
}

func getProfileByID(ctx context.Context, db profileGetter, profileID int64) (dbtypes.ModProfile, error) {
	var profile dbtypes.ModProfile
	err := db.GetContext(ctx, &profile, `
		SELECT id, game_id, name, created_at, updated_at
		FROM profiles
		WHERE id = ?
	`, profileID)
	if err != nil {
		return dbtypes.ModProfile{}, err
	}

	return profile, nil
}

func normalizeProfileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrProfileNameRequired
	}

	return name, nil
}
