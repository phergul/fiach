package storage

import (
	"context"
	"errors"
	"fmt"
)

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
		SELECT id, name, install_path, source, COALESCE(source_id, '') AS source_id, available, COALESCE(last_seen_at, '') AS last_seen_at
		FROM games
		WHERE available = 1
		ORDER BY LOWER(name), id
	`)
	if err != nil {
		return nil, err
	}

	return games, nil
}
