package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Mod struct {
	ID         int64  `db:"id"`
	GameID     int64  `db:"game_id"`
	Name       string `db:"name"`
	SourcePath string `db:"source_path"`
	CreatedAt  string `db:"created_at"`
	UpdatedAt  string `db:"updated_at"`
}

type ProfileMod struct {
	ProfileID  int64  `db:"profile_id"`
	ModID      int64  `db:"mod_id"`
	Name       string `db:"name"`
	SourcePath string `db:"source_path"`
	Enabled    bool   `db:"enabled"`
	LoadOrder  int64  `db:"load_order"`
	CreatedAt  string `db:"created_at"`
	UpdatedAt  string `db:"updated_at"`
}

func (s *Store) ListMods(ctx context.Context, gameID int64) (mods []Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select game mods: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}

	err = s.db.SelectContext(ctx, &mods, `
		SELECT id, game_id, name, source_path, created_at, updated_at
		FROM mods
		WHERE game_id = ?
		ORDER BY LOWER(name), id
	`, gameID)
	if err != nil {
		return nil, err
	}

	return mods, nil
}

func (s *Store) ListProfileMods(ctx context.Context, profileID int64) (mods []ProfileMod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select profile mods: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}

	err = s.db.SelectContext(ctx, &mods, profileModsSelectSQL+`
		WHERE pm.profile_id = ?
		ORDER BY pm.load_order, LOWER(m.name), m.id
	`, profileID)
	if err != nil {
		return nil, err
	}

	return mods, nil
}

func (s *Store) AddModToProfile(ctx context.Context, profileID int64, modID int64) (profileMod ProfileMod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("insert profile mod row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return ProfileMod{}, errors.New("store is not open")
	}

	err = withTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		existing, found, err := getProfileMod(ctx, tx, profileID, modID)
		if err != nil {
			return err
		}
		if found {
			profileMod = existing
			return nil
		}

		profile, err := getProfileByID(ctx, tx, profileID)
		if err != nil {
			return err
		}

		mod, err := getModByID(ctx, tx, modID)
		if err != nil {
			return err
		}
		if profile.GameID != mod.GameID {
			return fmt.Errorf("mod %d does not belong to profile game %d", modID, profile.GameID)
		}

		var loadOrder int64
		if err := tx.GetContext(ctx, &loadOrder, `
			SELECT COALESCE(MAX(load_order), -1) + 1
			FROM profile_mods
			WHERE profile_id = ?
		`, profileID); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO profile_mods (profile_id, mod_id, enabled, load_order)
			VALUES (?, ?, 1, ?)
		`, profileID, modID, loadOrder); err != nil {
			return err
		}

		profileMod, found, err = getProfileMod(ctx, tx, profileID, modID)
		if err != nil {
			return err
		}
		if !found {
			return sql.ErrNoRows
		}

		return nil
	})
	if err != nil {
		return ProfileMod{}, err
	}

	return profileMod, nil
}

func (s *Store) RemoveModFromProfile(ctx context.Context, profileID int64, modID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("delete profile mod row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}

	_, err = s.db.ExecContext(ctx, `
		DELETE FROM profile_mods
		WHERE profile_id = ?
			AND mod_id = ?
	`, profileID, modID)
	return err
}

func (s *Store) SetProfileModEnabled(ctx context.Context, profileID int64, modID int64, enabled bool) (profileMod ProfileMod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update profile mod enabled: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return ProfileMod{}, errors.New("store is not open")
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE profile_mods
		SET enabled = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE profile_id = ?
			AND mod_id = ?
	`, enabled, profileID, modID)
	if err != nil {
		return ProfileMod{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return ProfileMod{}, fmt.Errorf("get updated profile mod count: %w", err)
	}
	if affected == 0 {
		return ProfileMod{}, sql.ErrNoRows
	}

	profileMod, found, err := getProfileMod(ctx, s.db, profileID, modID)
	if err != nil {
		return ProfileMod{}, err
	}
	if !found {
		return ProfileMod{}, sql.ErrNoRows
	}

	return profileMod, nil
}

const profileModsSelectSQL = `
	SELECT
		pm.profile_id,
		pm.mod_id,
		m.name,
		m.source_path,
		pm.enabled,
		pm.load_order,
		pm.created_at,
		pm.updated_at
	FROM profile_mods pm
	INNER JOIN mods m ON m.id = pm.mod_id
`

type modGetter interface {
	GetContext(context.Context, any, string, ...any) error
}

func getModByID(ctx context.Context, db modGetter, modID int64) (Mod, error) {
	var mod Mod
	err := db.GetContext(ctx, &mod, `
		SELECT id, game_id, name, source_path, created_at, updated_at
		FROM mods
		WHERE id = ?
	`, modID)
	if err != nil {
		return Mod{}, err
	}

	return mod, nil
}

func getProfileMod(ctx context.Context, db modGetter, profileID int64, modID int64) (ProfileMod, bool, error) {
	var profileMod ProfileMod
	err := db.GetContext(ctx, &profileMod, profileModsSelectSQL+`
		WHERE pm.profile_id = ?
			AND pm.mod_id = ?
	`, profileID, modID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProfileMod{}, false, nil
		}

		return ProfileMod{}, false, err
	}

	return profileMod, true, nil
}
