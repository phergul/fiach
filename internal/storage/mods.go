package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/phergul/mod-manager/internal/storage/dbtypes"
)

func (s *Store) ListMods(ctx context.Context, gameID int64) (mods []dbtypes.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select game mods: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}

	err = s.db.SelectContext(ctx, &mods, `
		SELECT id, game_id, name, source_type, source_path, original_source_path, original_source_name, created_at, updated_at
		FROM mods
		WHERE game_id = ?
		ORDER BY LOWER(name), id
	`, gameID)
	if err != nil {
		return nil, err
	}

	return mods, nil
}

func (s *Store) GetMod(ctx context.Context, modID int64) (mod dbtypes.Mod, found bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select mod row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.Mod{}, false, errors.New("store is not open")
	}
	if modID <= 0 {
		return dbtypes.Mod{}, false, errors.New("mod ID must be positive")
	}

	mod, err = getModByID(ctx, s.db, modID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dbtypes.Mod{}, false, nil
		}

		return dbtypes.Mod{}, false, err
	}

	return mod, true, nil
}

func (s *Store) CreateMod(ctx context.Context, input dbtypes.CreateModInput) (mod dbtypes.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("insert mod row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.Mod{}, errors.New("store is not open")
	}

	mod, err = insertMod(ctx, s.db, input)
	if err != nil {
		return dbtypes.Mod{}, err
	}

	return mod, nil
}

func (s *Store) DeleteMod(ctx context.Context, modID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("delete mod row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}
	if modID <= 0 {
		return errors.New("mod ID must be positive")
	}

	result, err := s.db.ExecContext(ctx, `
		DELETE FROM mods
		WHERE id = ?
	`, modID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted mod count: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (s *Store) CountProfilesUsingMod(ctx context.Context, modID int64) (count int64, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("count profiles using mod: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return 0, errors.New("store is not open")
	}
	if modID <= 0 {
		return 0, errors.New("mod ID must be positive")
	}

	err = s.db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM profile_mods
		WHERE mod_id = ?
	`, modID)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Store) ProfileUsesMod(ctx context.Context, profileID int64, modID int64) (uses bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("check profile mod membership: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return false, errors.New("store is not open")
	}
	if profileID <= 0 {
		return false, errors.New("profile ID must be positive")
	}
	if modID <= 0 {
		return false, errors.New("mod ID must be positive")
	}

	var count int
	err = s.db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM profile_mods
		WHERE profile_id = ?
			AND mod_id = ?
	`, profileID, modID)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *Store) FindModByOriginalSourcePath(ctx context.Context, gameID int64, originalSourcePath string) (mod dbtypes.Mod, found bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("find mod by original source path: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.Mod{}, false, errors.New("store is not open")
	}

	originalSourcePath, err = CanonicalModOriginalSourcePath(originalSourcePath)
	if err != nil {
		return dbtypes.Mod{}, false, err
	}

	err = s.db.GetContext(ctx, &mod, `
		SELECT id, game_id, name, source_type, source_path, original_source_path, original_source_name, created_at, updated_at
		FROM mods
		WHERE game_id = ?
			AND original_source_path = ?
	`, gameID, originalSourcePath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dbtypes.Mod{}, false, nil
		}

		return dbtypes.Mod{}, false, err
	}

	return mod, true, nil
}

func insertMod(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	GetContext(context.Context, any, string, ...any) error
}, input dbtypes.CreateModInput) (dbtypes.Mod, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return dbtypes.Mod{}, errors.New("mod name is required")
	}

	sourceType := input.SourceType
	if sourceType == "" {
		sourceType = dbtypes.ModSourceTypeFolder
	}
	if sourceType != dbtypes.ModSourceTypeFolder && sourceType != dbtypes.ModSourceTypeArchive {
		return dbtypes.Mod{}, fmt.Errorf("unsupported mod source type %q", sourceType)
	}

	sourcePath := cleanOptionalPath(input.SourcePath)
	if sourcePath == "" {
		return dbtypes.Mod{}, errors.New("managed mod source path is required")
	}

	originalSourcePath, err := CanonicalModOriginalSourcePath(input.OriginalSourcePath)
	if err != nil {
		return dbtypes.Mod{}, err
	}

	originalSourceName := cleanOptionalString(input.OriginalSourceName)
	result, err := db.ExecContext(ctx, `
		INSERT INTO mods (game_id, name, source_type, source_path, original_source_path, original_source_name)
		VALUES (?, ?, ?, ?, ?, ?)
	`, input.GameID, name, sourceType, sourcePath, originalSourcePath, nullableText(originalSourceName))
	if err != nil {
		return dbtypes.Mod{}, err
	}

	modID, err := result.LastInsertId()
	if err != nil {
		return dbtypes.Mod{}, fmt.Errorf("get created mod id: %w", err)
	}

	return getModByID(ctx, db, modID)
}

type modGetter interface {
	GetContext(context.Context, any, string, ...any) error
}

func getModByID(ctx context.Context, db modGetter, modID int64) (dbtypes.Mod, error) {
	var mod dbtypes.Mod
	err := db.GetContext(ctx, &mod, `
		SELECT id, game_id, name, source_type, source_path, original_source_path, original_source_name, created_at, updated_at
		FROM mods
		WHERE id = ?
	`, modID)
	if err != nil {
		return dbtypes.Mod{}, err
	}

	return mod, nil
}

func CanonicalModOriginalSourcePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("original source path is required")
	}

	path = filepath.Clean(path)
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return path, nil
	}

	return filepath.Clean(absolutePath), nil
}
