package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func (s *Store) ReplaceAppliedCreatedDirectories(ctx context.Context, input dbtypes.ReplaceAppliedCreatedDirectoriesInput) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("replace applied created directories: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}
	if err := validateReplaceAppliedCreatedDirectoriesInput(input); err != nil {
		return err
	}

	return withTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		return replaceAppliedCreatedDirectories(ctx, tx, input)
	})
}

func (s *Store) ListAppliedCreatedDirectories(ctx context.Context, gameID int64) (directories []dbtypes.AppliedCreatedDirectoryRow, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list applied created directories: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}
	if gameID <= 0 {
		return nil, errors.New("game ID must be positive")
	}

	err = s.db.SelectContext(ctx, &directories, `
		SELECT
			game_id,
			game_relative_path,
			mod_id,
			mod_name
		FROM applied_created_directories
		WHERE game_id = ?
		ORDER BY game_relative_path COLLATE NOCASE
	`, gameID)
	if err != nil {
		return nil, err
	}
	if directories == nil {
		directories = []dbtypes.AppliedCreatedDirectoryRow{}
	}

	return directories, nil
}

func replaceAppliedCreatedDirectories(ctx context.Context, tx *sqlx.Tx, input dbtypes.ReplaceAppliedCreatedDirectoriesInput) error {
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM applied_created_directories
		WHERE game_id = ?
	`, input.GameID); err != nil {
		return err
	}

	for _, row := range input.Directories {
		if err := insertAppliedCreatedDirectoryRow(ctx, tx, row); err != nil {
			return err
		}
	}

	return nil
}

func insertAppliedCreatedDirectoryRow(ctx context.Context, tx *sqlx.Tx, row dbtypes.AppliedCreatedDirectoryRow) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO applied_created_directories (
			game_id,
			game_relative_path,
			mod_id,
			mod_name
		)
		VALUES (?, ?, ?, ?)
	`, row.GameID, row.GameRelativePath, nullableInt64(row.ModID), nullableText(stringValue(row.ModName))); err != nil {
		return err
	}

	return nil
}

func validateReplaceAppliedCreatedDirectoriesInput(input dbtypes.ReplaceAppliedCreatedDirectoriesInput) error {
	if input.GameID <= 0 {
		return errors.New("game ID must be positive")
	}

	for _, row := range input.Directories {
		if row.GameID != input.GameID {
			return errors.New("created directory game ID must match input game ID")
		}
		if strings.TrimSpace(row.GameRelativePath) == "" {
			return errors.New("created directory game relative path is required")
		}
	}

	return nil
}
