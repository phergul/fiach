package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/phergul/mod-manager/internal/storage/dbtypes"
)

const gameModsDirName = "mods"

func (s *Store) ListStoredGames(ctx context.Context) (games []dbtypes.StoredGame, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list stored games: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}

	err = s.db.SelectContext(ctx, &games, `
		SELECT id, name, install_path, source, source_id, available, last_seen_at, mod_storage_path, mod_storage_path_override
		FROM games
		WHERE available = 1
		ORDER BY LOWER(name), id
	`)
	if err != nil {
		return nil, err
	}

	return games, nil
}

func (s *Store) GetStoredGame(ctx context.Context, gameID int64) (game dbtypes.StoredGame, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get stored game: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.StoredGame{}, errors.New("store is not open")
	}

	err = s.db.GetContext(ctx, &game, `
		SELECT id, name, install_path, source, source_id, available, last_seen_at, mod_storage_path, mod_storage_path_override
		FROM games
		WHERE id = ?
	`, gameID)
	if err != nil {
		return dbtypes.StoredGame{}, err
	}

	return game, nil
}

func (s *Store) SetGameModStoragePathOverride(ctx context.Context, gameID int64, path string) (game dbtypes.StoredGame, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update game mod storage path override: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.StoredGame{}, errors.New("store is not open")
	}

	globalRoot, err := s.GetGlobalModStorageRoot(ctx)
	if err != nil {
		return dbtypes.StoredGame{}, fmt.Errorf("read managed mod storage root: %w", err)
	}

	path = cleanOptionalPath(path)
	modStoragePath := path
	if modStoragePath == "" {
		game, err = s.GetStoredGame(ctx, gameID)
		if err != nil {
			return dbtypes.StoredGame{}, err
		}
		game.ModStoragePathOverride = nil
		modStoragePath, err = resolveStoredGameModStoragePath(game, globalRoot, s.defaultModStorageRoot())
		if err != nil {
			return dbtypes.StoredGame{}, fmt.Errorf("resolve cleared game mod storage path: %w", err)
		}
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE games
		SET mod_storage_path_override = ?,
			mod_storage_path = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, nullableText(path), modStoragePath, gameID)
	if err != nil {
		return dbtypes.StoredGame{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return dbtypes.StoredGame{}, fmt.Errorf("get updated game count: %w", err)
	}
	if affected == 0 {
		return dbtypes.StoredGame{}, sql.ErrNoRows
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

	defaultRoot := s.defaultModStorageRoot()
	if defaultRoot == "" {
		return "", fmt.Errorf("default mod storage root is not configured")
	}

	return resolveStoredGameModStoragePath(game, globalRoot, defaultRoot)
}

func DefaultGameModStorageFolderName(game dbtypes.StoredGame) string {
	return strconv.FormatInt(game.ID, 10)
}

func (s *Store) defaultModStorageRoot() string {
	if s == nil || s.path == "" {
		return ""
	}

	return filepath.Join(filepath.Dir(s.path), gameModsDirName)
}

func resolveStoredGameModStoragePath(game dbtypes.StoredGame, globalRoot string, defaultRoot string) (string, error) {
	override := cleanOptionalStringPath(game.ModStoragePathOverride)
	if override != "" {
		return override, nil
	}

	root := cleanOptionalPath(globalRoot)
	if root == "" {
		root = cleanOptionalPath(defaultRoot)
	}
	if root == "" {
		return "", errors.New("managed mod storage root is not configured")
	}

	folderName := DefaultGameModStorageFolderName(game)
	if folderName == "" {
		return "", errors.New("game ID is required for managed mod storage path")
	}

	return filepath.Join(root, folderName), nil
}

func (s *Store) RefreshGameModStoragePaths(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("refresh game mod storage paths: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}

	globalRoot, err := s.GetGlobalModStorageRoot(ctx)
	if err != nil {
		return err
	}

	games := []dbtypes.StoredGame{}
	if err := s.db.SelectContext(ctx, &games, `
		SELECT id, name, install_path, source, source_id, available, last_seen_at, mod_storage_path, mod_storage_path_override
		FROM games
	`); err != nil {
		return err
	}

	defaultRoot := s.defaultModStorageRoot()
	for _, game := range games {
		path, err := resolveStoredGameModStoragePath(game, globalRoot, defaultRoot)
		if err != nil {
			return fmt.Errorf("resolve game %d mod storage path: %w", game.ID, err)
		}

		if _, err := s.db.ExecContext(ctx, `
			UPDATE games
			SET mod_storage_path = ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, path, game.ID); err != nil {
			return fmt.Errorf("update game %d mod storage path: %w", game.ID, err)
		}
	}

	return nil
}
