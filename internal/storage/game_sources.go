package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func getStoredGameBySource(ctx context.Context, tx *sqlx.Tx, source string, sourceID string) (dbtypes.StoredGame, bool, error) {
	var game dbtypes.StoredGame
	err := tx.GetContext(ctx, &game, `
		SELECT id, name, install_path, source, source_id, available, last_seen_at, mod_storage_path, mod_storage_path_override
		FROM games
		WHERE source = ?
			AND source_id = ?
	`, source, sourceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return dbtypes.StoredGame{}, false, nil
		}

		return dbtypes.StoredGame{}, false, fmt.Errorf("find game by source: %w", err)
	}

	return game, true, nil
}

func getStoredGameByInstallPath(ctx context.Context, tx *sqlx.Tx, installPath string) (dbtypes.StoredGame, bool, error) {
	var game dbtypes.StoredGame
	err := tx.GetContext(ctx, &game, `
		SELECT id, name, install_path, source, source_id, available, last_seen_at, mod_storage_path, mod_storage_path_override
		FROM games
		WHERE install_path = ?
	`, installPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return dbtypes.StoredGame{}, false, nil
		}

		return dbtypes.StoredGame{}, false, fmt.Errorf("find game by install path: %w", err)
	}

	return game, true, nil
}

func getStoredGameByID(ctx context.Context, tx *sqlx.Tx, id int64) (dbtypes.StoredGame, error) {
	var game dbtypes.StoredGame
	err := tx.GetContext(ctx, &game, `
		SELECT id, name, install_path, source, source_id, available, last_seen_at, mod_storage_path, mod_storage_path_override
		FROM games
		WHERE id = ?
	`, id)
	if err != nil {
		return dbtypes.StoredGame{}, fmt.Errorf("get stored game: %w", err)
	}

	return game, nil
}
