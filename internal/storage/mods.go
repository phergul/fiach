package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type Mod struct {
	ID                 int64         `db:"id"`
	GameID             int64         `db:"game_id"`
	Name               string        `db:"name"`
	SourceType         ModSourceType `db:"source_type"`
	SourcePath         string        `db:"source_path"`
	OriginalSourcePath string        `db:"original_source_path"`
	OriginalSourceName *string       `db:"original_source_name"`
	CreatedAt          string        `db:"created_at"`
	UpdatedAt          string        `db:"updated_at"`
}

type ModSourceType string

const (
	ModSourceTypeFolder  ModSourceType = "folder"
	ModSourceTypeArchive ModSourceType = "archive"
)

type CreateModInput struct {
	GameID             int64
	Name               string
	SourceType         ModSourceType
	SourcePath         string
	OriginalSourcePath string
	OriginalSourceName *string
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

func (s *Store) CreateMod(ctx context.Context, input CreateModInput) (mod Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("insert mod row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return Mod{}, errors.New("store is not open")
	}

	mod, err = insertMod(ctx, s.db, input)
	if err != nil {
		return Mod{}, err
	}

	return mod, nil
}

func (s *Store) FindModByOriginalSourcePath(ctx context.Context, gameID int64, originalSourcePath string) (mod Mod, found bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("find mod by original source path: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return Mod{}, false, errors.New("store is not open")
	}

	originalSourcePath, err = CanonicalModOriginalSourcePath(originalSourcePath)
	if err != nil {
		return Mod{}, false, err
	}

	err = s.db.GetContext(ctx, &mod, `
		SELECT id, game_id, name, source_type, source_path, original_source_path, original_source_name, created_at, updated_at
		FROM mods
		WHERE game_id = ?
			AND original_source_path = ?
	`, gameID, originalSourcePath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Mod{}, false, nil
		}

		return Mod{}, false, err
	}

	return mod, true, nil
}

func insertMod(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	GetContext(context.Context, any, string, ...any) error
}, input CreateModInput) (Mod, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Mod{}, errors.New("mod name is required")
	}

	sourceType := input.SourceType
	if sourceType == "" {
		sourceType = ModSourceTypeFolder
	}
	if sourceType != ModSourceTypeFolder && sourceType != ModSourceTypeArchive {
		return Mod{}, fmt.Errorf("unsupported mod source type %q", sourceType)
	}

	sourcePath := cleanOptionalPath(input.SourcePath)
	if sourcePath == "" {
		return Mod{}, errors.New("managed mod source path is required")
	}

	originalSourcePath, err := CanonicalModOriginalSourcePath(input.OriginalSourcePath)
	if err != nil {
		return Mod{}, err
	}

	originalSourceName := cleanOptionalString(input.OriginalSourceName)
	result, err := db.ExecContext(ctx, `
		INSERT INTO mods (game_id, name, source_type, source_path, original_source_path, original_source_name)
		VALUES (?, ?, ?, ?, ?, ?)
	`, input.GameID, name, sourceType, sourcePath, originalSourcePath, nullableText(originalSourceName))
	if err != nil {
		return Mod{}, err
	}

	modID, err := result.LastInsertId()
	if err != nil {
		return Mod{}, fmt.Errorf("get created mod id: %w", err)
	}

	return getModByID(ctx, db, modID)
}

type modGetter interface {
	GetContext(context.Context, any, string, ...any) error
}

func getModByID(ctx context.Context, db modGetter, modID int64) (Mod, error) {
	var mod Mod
	err := db.GetContext(ctx, &mod, `
		SELECT id, game_id, name, source_type, source_path, original_source_path, original_source_name, created_at, updated_at
		FROM mods
		WHERE id = ?
	`, modID)
	if err != nil {
		return Mod{}, err
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
