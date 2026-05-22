package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

const (
	GameSourceManual = "manual"
	GameSourceSteam  = "steam"
)

type StoredGame struct {
	ID                     int64   `db:"id"`
	Name                   string  `db:"name"`
	InstallPath            string  `db:"install_path"`
	Source                 string  `db:"source"`
	SourceID               *string `db:"source_id"`
	Available              bool    `db:"available"`
	LastSeenAt             *string `db:"last_seen_at"`
	ModStoragePath         *string `db:"mod_storage_path"`
	ModStoragePathOverride *string `db:"mod_storage_path_override"`
}

type SourceGame struct {
	SourceID    string
	Name        string
	InstallPath string
}

func getStoredGameBySource(ctx context.Context, tx *sqlx.Tx, source string, sourceID string) (StoredGame, bool, error) {
	var game StoredGame
	err := tx.GetContext(ctx, &game, `
		SELECT id, name, install_path, source, source_id, available, last_seen_at, mod_storage_path, mod_storage_path_override
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
		SELECT id, name, install_path, source, source_id, available, last_seen_at, mod_storage_path, mod_storage_path_override
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

func getStoredGameByID(ctx context.Context, tx *sqlx.Tx, id int64) (StoredGame, error) {
	var game StoredGame
	err := tx.GetContext(ctx, &game, `
		SELECT id, name, install_path, source, source_id, available, last_seen_at, mod_storage_path, mod_storage_path_override
		FROM games
		WHERE id = ?
	`, id)
	if err != nil {
		return StoredGame{}, fmt.Errorf("get stored game: %w", err)
	}

	return game, nil
}
