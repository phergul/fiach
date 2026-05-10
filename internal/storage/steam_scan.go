package storage

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/phergul/mod-manager/internal/steam"
)

const (
	GameSourceManual = "manual"
	GameSourceSteam  = "steam"
)

type StoredGame struct {
	ID          int64  `db:"id"`
	Name        string `db:"name"`
	InstallPath string `db:"install_path"`
	Source      string `db:"source"`
	SourceID    string `db:"source_id"`
	Available   bool   `db:"available"`
	LastSeenAt  string `db:"last_seen_at"`
}

type SteamScanResult struct {
	Inserted          int
	Updated           int
	MarkedUnavailable int
	Games             []StoredGame
}

func (s *Store) SaveSteamScan(ctx context.Context, games []steam.Game) (SteamScanResult, error) {
	var result SteamScanResult
	if s == nil || s.db == nil {
		return result, fmt.Errorf("save Steam scan: store is not open")
	}

	err := withTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		seenIDs := make([]string, 0, len(games))
		seenSet := make(map[string]struct{}, len(games))
		now := time.Now().UTC().Format(time.RFC3339)

		for _, game := range dedupeSteamGames(games) {
			stored, action, err := upsertSteamGame(ctx, tx, game, now)
			if err != nil {
				return err
			}

			switch action {
			case "inserted":
				result.Inserted++
			case "updated":
				result.Updated++
			}

			if _, ok := seenSet[stored.SourceID]; !ok {
				seenIDs = append(seenIDs, stored.SourceID)
				seenSet[stored.SourceID] = struct{}{}
			}
			result.Games = append(result.Games, stored)
		}

		markedUnavailable, err := markUnavailableSteamGames(ctx, tx, seenIDs)
		if err != nil {
			return err
		}
		result.MarkedUnavailable = markedUnavailable

		return nil
	})
	if err != nil {
		return SteamScanResult{}, fmt.Errorf("save Steam scan: %w", err)
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

func upsertSteamGame(ctx context.Context, tx *sqlx.Tx, game steam.Game, seenAt string) (StoredGame, string, error) {
	sourceID := strings.TrimSpace(game.AppID)
	name := strings.TrimSpace(game.Name)
	installPath := filepath.Clean(strings.TrimSpace(game.InstallPath))
	if sourceID == "" || name == "" || installPath == "." {
		return StoredGame{}, "", fmt.Errorf("Steam game is missing app ID, name, or install path")
	}

	stored, found, err := getStoredGameBySource(ctx, tx, GameSourceSteam, sourceID)
	if err != nil {
		return StoredGame{}, "", err
	}

	action := "updated"
	if !found {
		stored, found, err = getStoredGameByInstallPath(ctx, tx, installPath)
		if err != nil {
			return StoredGame{}, "", err
		}
	}

	if found {
		updated, err := updateStoredGameAsSteam(ctx, tx, stored.ID, name, installPath, sourceID, seenAt)
		if err != nil {
			return StoredGame{}, "", err
		}

		return updated, action, nil
	}

	inserted, err := insertSteamGame(ctx, tx, name, installPath, sourceID, seenAt)
	if err != nil {
		return StoredGame{}, "", err
	}

	return inserted, "inserted", nil
}

func getStoredGameBySource(ctx context.Context, tx *sqlx.Tx, source string, sourceID string) (StoredGame, bool, error) {
	var game StoredGame
	err := tx.GetContext(ctx, &game, `
		SELECT id, name, install_path, source, COALESCE(source_id, '') AS source_id, available, COALESCE(last_seen_at, '') AS last_seen_at
		FROM games
		WHERE source = ?
			AND source_id = ?
	`, source, sourceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return StoredGame{}, false, nil
		}

		return StoredGame{}, false, fmt.Errorf("find game by source: %w", err)
	}

	return game, true, nil
}

func getStoredGameByInstallPath(ctx context.Context, tx *sqlx.Tx, installPath string) (StoredGame, bool, error) {
	var game StoredGame
	err := tx.GetContext(ctx, &game, `
		SELECT id, name, install_path, source, COALESCE(source_id, '') AS source_id, available, COALESCE(last_seen_at, '') AS last_seen_at
		FROM games
		WHERE install_path = ?
	`, installPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return StoredGame{}, false, nil
		}

		return StoredGame{}, false, fmt.Errorf("find game by install path: %w", err)
	}

	return game, true, nil
}

func updateStoredGameAsSteam(ctx context.Context, tx *sqlx.Tx, id int64, name string, installPath string, sourceID string, seenAt string) (StoredGame, error) {
	_, err := tx.ExecContext(ctx, `
		UPDATE games
		SET name = ?,
			install_path = ?,
			source = ?,
			source_id = ?,
			available = 1,
			last_seen_at = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, installPath, GameSourceSteam, sourceID, seenAt, id)
	if err != nil {
		return StoredGame{}, fmt.Errorf("update Steam game: %w", err)
	}

	return getStoredGameByID(ctx, tx, id)
}

func insertSteamGame(ctx context.Context, tx *sqlx.Tx, name string, installPath string, sourceID string, seenAt string) (StoredGame, error) {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO games (name, install_path, source, source_id, available, last_seen_at)
		VALUES (?, ?, ?, ?, 1, ?)
	`, name, installPath, GameSourceSteam, sourceID, seenAt)
	if err != nil {
		return StoredGame{}, fmt.Errorf("insert Steam game: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return StoredGame{}, fmt.Errorf("insert Steam game id: %w", err)
	}

	return getStoredGameByID(ctx, tx, id)
}

func getStoredGameByID(ctx context.Context, tx *sqlx.Tx, id int64) (StoredGame, error) {
	var game StoredGame
	err := tx.GetContext(ctx, &game, `
		SELECT id, name, install_path, source, COALESCE(source_id, '') AS source_id, available, COALESCE(last_seen_at, '') AS last_seen_at
		FROM games
		WHERE id = ?
	`, id)
	if err != nil {
		return StoredGame{}, fmt.Errorf("get stored game: %w", err)
	}

	return game, nil
}

func markUnavailableSteamGames(ctx context.Context, tx *sqlx.Tx, seenIDs []string) (int, error) {
	query := `
		UPDATE games
		SET available = 0,
			updated_at = CURRENT_TIMESTAMP
		WHERE source = ?
			AND available = 1
	`
	args := []any{GameSourceSteam}
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
		return 0, fmt.Errorf("mark unavailable Steam games: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("mark unavailable Steam games count: %w", err)
	}

	return int(count), nil
}

func dedupeSteamGames(games []steam.Game) []steam.Game {
	seen := make(map[string]struct{}, len(games))
	result := make([]steam.Game, 0, len(games))

	for _, game := range games {
		appID := strings.TrimSpace(game.AppID)
		if appID == "" {
			continue
		}
		if _, ok := seen[appID]; ok {
			continue
		}

		seen[appID] = struct{}{}
		result = append(result, game)
	}

	return result
}
