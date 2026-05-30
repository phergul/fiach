package storage

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func (s *Store) SaveSourceScan(ctx context.Context, source string, games []dbtypes.SourceGame) (result dbtypes.SourceScanResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("save source scan: %w", err)
		}
	}()

	result, err = s.saveSourceScan(ctx, source, games)
	if err != nil {
		return dbtypes.SourceScanResult{}, err
	}

	return result, nil
}

func (s *Store) saveSourceScan(ctx context.Context, source string, games []dbtypes.SourceGame) (result dbtypes.SourceScanResult, err error) {
	if s == nil || s.db == nil {
		return result, errors.New("store is not open")
	}

	source = strings.TrimSpace(source)
	if source == "" {
		return result, errors.New("source is required")
	}

	globalRoot, err := s.GetGlobalModStorageRoot(ctx)
	if err != nil {
		return dbtypes.SourceScanResult{}, fmt.Errorf("read managed mod storage root: %w", err)
	}
	defaultRoot := s.defaultModStorageRoot()

	err = withTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		seenIDs := make([]string, 0, len(games))
		seenSet := make(map[string]struct{}, len(games))
		now := time.Now().UTC().Format(time.RFC3339)

		for _, game := range dedupeSourceGames(games) {
			stored, action, err := upsertSourceGame(ctx, tx, source, game, now, globalRoot, defaultRoot)
			if err != nil {
				return err
			}

			switch action {
			case "inserted":
				result.Inserted++
			case "updated":
				result.Updated++
			}

			sourceID := strings.TrimSpace(game.SourceID)
			if _, ok := seenSet[sourceID]; !ok {
				seenIDs = append(seenIDs, sourceID)
				seenSet[sourceID] = struct{}{}
			}
			result.Games = append(result.Games, stored)
		}

		markedUnavailable, err := markUnavailableSourceGames(ctx, tx, source, seenIDs)
		if err != nil {
			return err
		}
		result.MarkedUnavailable = markedUnavailable

		return nil
	})
	if err != nil {
		return dbtypes.SourceScanResult{}, err
	}

	return result, nil
}

func withTransaction(ctx context.Context, db *sqlx.DB, fn func(*sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func upsertSourceGame(ctx context.Context, tx *sqlx.Tx, source string, game dbtypes.SourceGame, seenAt string, globalRoot string, defaultRoot string) (dbtypes.StoredGame, string, error) {
	sourceID := strings.TrimSpace(game.SourceID)
	name := strings.TrimSpace(game.Name)
	installPath := filepath.Clean(strings.TrimSpace(game.InstallPath))
	if sourceID == "" || name == "" || installPath == "." {
		return dbtypes.StoredGame{}, "", fmt.Errorf("source game is missing source ID, name, or install path")
	}

	stored, found, err := getStoredGameBySource(ctx, tx, source, sourceID)
	if err != nil {
		return dbtypes.StoredGame{}, "", err
	}

	action := "updated"
	if !found {
		stored, found, err = getStoredGameByInstallPath(ctx, tx, installPath)
		if err != nil {
			return dbtypes.StoredGame{}, "", err
		}
	}

	if found {
		updated, err := updateStoredGameSource(ctx, tx, stored.ID, name, installPath, source, sourceID, seenAt, globalRoot, defaultRoot)
		if err != nil {
			return dbtypes.StoredGame{}, "", err
		}

		return updated, action, nil
	}

	inserted, err := insertSourceGame(ctx, tx, name, installPath, source, sourceID, seenAt, globalRoot, defaultRoot)
	if err != nil {
		return dbtypes.StoredGame{}, "", err
	}

	return inserted, "inserted", nil
}

func updateStoredGameSource(ctx context.Context, tx *sqlx.Tx, id int64, name string, installPath string, source string, sourceID string, seenAt string, globalRoot string, defaultRoot string) (dbtypes.StoredGame, error) {
	stored, err := getStoredGameByID(ctx, tx, id)
	if err != nil {
		return dbtypes.StoredGame{}, err
	}
	stored.Name = name
	modStoragePath, err := resolveStoredGameModStoragePath(stored, globalRoot, defaultRoot)
	if err != nil {
		return dbtypes.StoredGame{}, fmt.Errorf("resolve source game mod storage path: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE games
		SET name = ?,
			install_path = ?,
			source = ?,
			source_id = ?,
			available = 1,
			last_seen_at = ?,
			mod_storage_path = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, installPath, source, sourceID, seenAt, modStoragePath, id)
	if err != nil {
		return dbtypes.StoredGame{}, fmt.Errorf("update source game: %w", err)
	}

	return getStoredGameByID(ctx, tx, id)
}

func insertSourceGame(ctx context.Context, tx *sqlx.Tx, name string, installPath string, source string, sourceID string, seenAt string, globalRoot string, defaultRoot string) (dbtypes.StoredGame, error) {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO games (name, install_path, source, source_id, available, last_seen_at)
		VALUES (?, ?, ?, ?, 1, ?)
	`, name, installPath, source, sourceID, seenAt)
	if err != nil {
		return dbtypes.StoredGame{}, fmt.Errorf("insert source game: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return dbtypes.StoredGame{}, fmt.Errorf("insert source game id: %w", err)
	}

	stored := dbtypes.StoredGame{
		ID:   id,
		Name: name,
	}
	modStoragePath, err := resolveStoredGameModStoragePath(stored, globalRoot, defaultRoot)
	if err != nil {
		return dbtypes.StoredGame{}, fmt.Errorf("resolve source game mod storage path: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE games
		SET mod_storage_path = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, modStoragePath, id); err != nil {
		return dbtypes.StoredGame{}, fmt.Errorf("update source game mod storage path: %w", err)
	}

	return getStoredGameByID(ctx, tx, id)
}

func markUnavailableSourceGames(ctx context.Context, tx *sqlx.Tx, source string, seenIDs []string) (int, error) {
	query := `
		UPDATE games
		SET available = 0,
			updated_at = CURRENT_TIMESTAMP
		WHERE source = ?
			AND available = 1
	`
	args := []any{source}
	if len(seenIDs) > 0 {
		query += ` AND source_id NOT IN (?)`
		var err error
		query, args, err = sqlx.In(query, append(args, any(seenIDs))...)
		if err != nil {
			return 0, fmt.Errorf("build unavailable query: %w", err)
		}
		query = tx.Rebind(query)
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("mark unavailable source games: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("mark unavailable source games count: %w", err)
	}

	return int(count), nil
}

func dedupeSourceGames(games []dbtypes.SourceGame) []dbtypes.SourceGame {
	seen := make(map[string]struct{}, len(games))
	result := make([]dbtypes.SourceGame, 0, len(games))

	for _, game := range games {
		sourceID := strings.TrimSpace(game.SourceID)
		if sourceID == "" {
			continue
		}
		if _, ok := seen[sourceID]; ok {
			continue
		}

		game.SourceID = sourceID
		seen[sourceID] = struct{}{}
		result = append(result, game)
	}

	return result
}
