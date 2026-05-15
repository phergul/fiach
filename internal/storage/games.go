package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var unsafeStoragePathNameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]+`)
var repeatedStoragePathSeparators = regexp.MustCompile(`-+`)

func (s *Store) ListStoredGames(ctx context.Context) (games []StoredGame, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list stored games: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}

	err = s.db.SelectContext(ctx, &games, `
		SELECT id, name, install_path, source, COALESCE(source_id, '') AS source_id, available, COALESCE(last_seen_at, '') AS last_seen_at, mod_storage_path_override
		FROM games
		WHERE available = 1
		ORDER BY LOWER(name), id
	`)
	if err != nil {
		return nil, err
	}

	return games, nil
}

func (s *Store) GetStoredGame(ctx context.Context, gameID int64) (game StoredGame, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get stored game: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return StoredGame{}, errors.New("store is not open")
	}

	err = s.db.GetContext(ctx, &game, `
		SELECT id, name, install_path, source, COALESCE(source_id, '') AS source_id, available, COALESCE(last_seen_at, '') AS last_seen_at, mod_storage_path_override
		FROM games
		WHERE id = ?
	`, gameID)
	if err != nil {
		return StoredGame{}, err
	}

	return game, nil
}

func (s *Store) SetGameModStoragePathOverride(ctx context.Context, gameID int64, path string) (game StoredGame, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update game mod storage path override: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return StoredGame{}, errors.New("store is not open")
	}

	path = cleanOptionalPath(path)
	result, err := s.db.ExecContext(ctx, `
		UPDATE games
		SET mod_storage_path_override = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, nullablePath(path), gameID)
	if err != nil {
		return StoredGame{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return StoredGame{}, fmt.Errorf("get updated game count: %w", err)
	}
	if affected == 0 {
		return StoredGame{}, sql.ErrNoRows
	}

	return s.GetStoredGame(ctx, gameID)
}

func (s *Store) ResolveGameModStoragePath(ctx context.Context, gameID int64, globalRoot string) (path string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("build game mod storage path: %w", err)
		}
	}()

	game, err := s.GetStoredGame(ctx, gameID)
	if err != nil {
		return "", fmt.Errorf("select game for mod storage path: %w", err)
	}

	override := cleanOptionalStringPath(game.ModStoragePathOverride)
	if override != "" {
		return override, nil
	}

	globalRoot = cleanOptionalPath(globalRoot)
	if globalRoot == "" {
		return "", fmt.Errorf("global mod storage root is not configured")
	}

	return filepath.Join(globalRoot, DefaultGameModStorageFolderName(game)), nil
}

func DefaultGameModStorageFolderName(game StoredGame) string {
	name := strings.TrimSpace(game.Name)
	name = unsafeStoragePathNameChars.ReplaceAllString(name, "-")
	name = repeatedStoragePathSeparators.ReplaceAllString(name, "-")
	name = strings.Trim(name, " .-")
	if name == "" {
		name = "game"
	}

	return name + "-" + strconv.FormatInt(game.ID, 10)
}

func cleanOptionalPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	return filepath.Clean(path)
}

func nullablePath(path string) any {
	if path == "" {
		return nil
	}

	return path
}

func cleanOptionalStringPath(path *string) string {
	if path == nil {
		return ""
	}

	return cleanOptionalPath(*path)
}
