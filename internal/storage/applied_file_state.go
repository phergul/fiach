package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func (s *Store) ReplaceAppliedFileStates(ctx context.Context, input dbtypes.ReplaceAppliedFileStatesInput) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("replace applied file states: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}
	if err := validateReplaceAppliedFileStatesInput(input); err != nil {
		return err
	}

	return withTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		return replaceAppliedFileStates(ctx, tx, input)
	})
}

func (s *Store) ListAppliedFileStates(ctx context.Context, gameID int64) (states []dbtypes.AppliedFileStateRow, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list applied file states: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}
	if gameID <= 0 {
		return nil, errors.New("game ID must be positive")
	}

	err = s.db.SelectContext(ctx, &states, `
		SELECT
			game_id,
			game_relative_path,
			profile_id,
			baseline_exists,
			baseline_sha256,
			baseline_size_bytes,
			baseline_backup_path,
			applied_exists,
			applied_sha256,
			applied_size_bytes,
			winning_source_kind,
			winning_source_id,
			winning_mod_id,
			winning_load_order,
			output_kind,
			user_decision,
			last_applied_at
		FROM applied_file_states
		WHERE game_id = ?
		ORDER BY game_relative_path COLLATE NOCASE
	`, gameID)
	if err != nil {
		return nil, err
	}
	if states == nil {
		states = []dbtypes.AppliedFileStateRow{}
	}

	return states, nil
}

func (s *Store) HasAppliedFileStates(ctx context.Context, gameID int64) (found bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("check applied file states: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return false, errors.New("store is not open")
	}
	if gameID <= 0 {
		return false, errors.New("game ID must be positive")
	}

	var count int
	if err := s.db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM applied_file_states
		WHERE game_id = ?
	`, gameID); err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *Store) DeleteAppliedFileStates(ctx context.Context, gameID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("delete applied file states: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}
	if gameID <= 0 {
		return errors.New("game ID must be positive")
	}

	_, err = s.db.ExecContext(ctx, `
		DELETE FROM applied_file_states
		WHERE game_id = ?
	`, gameID)
	if err != nil {
		return err
	}

	return nil
}

func replaceAppliedFileStates(ctx context.Context, tx *sqlx.Tx, input dbtypes.ReplaceAppliedFileStatesInput) error {
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM applied_file_states
		WHERE game_id = ?
	`, input.GameID); err != nil {
		return err
	}

	for _, row := range input.FileStates {
		if err := insertAppliedFileStateRow(ctx, tx, row); err != nil {
			return err
		}
	}

	return nil
}

func insertAppliedFileStateRow(ctx context.Context, tx *sqlx.Tx, row dbtypes.AppliedFileStateRow) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO applied_file_states (
			game_id,
			game_relative_path,
			profile_id,
			baseline_exists,
			baseline_sha256,
			baseline_size_bytes,
			baseline_backup_path,
			applied_exists,
			applied_sha256,
			applied_size_bytes,
			winning_source_kind,
			winning_source_id,
			winning_mod_id,
			winning_load_order,
			output_kind,
			user_decision,
			last_applied_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, row.GameID, row.GameRelativePath, row.ProfileID, boolToInt(row.BaselineExists),
		nullableText(stringValue(row.BaselineSHA256)), nullableInt64(row.BaselineSizeBytes),
		nullableText(stringValue(row.BaselineBackupPath)), boolToInt(row.AppliedExists),
		nullableText(stringValue(row.AppliedSHA256)), nullableInt64(row.AppliedSizeBytes),
		nullableText(stringValue(row.WinningSourceKind)), nullableText(stringValue(row.WinningSourceID)),
		nullableInt64(row.WinningModID), nullableInt64(row.WinningLoadOrder),
		row.OutputKind, nullableText(stringValue(row.UserDecision)), row.LastAppliedAt); err != nil {
		return err
	}

	return nil
}

func validateReplaceAppliedFileStatesInput(input dbtypes.ReplaceAppliedFileStatesInput) error {
	if input.GameID <= 0 {
		return errors.New("game ID must be positive")
	}
	if input.ProfileID <= 0 {
		return errors.New("profile ID must be positive")
	}

	for _, row := range input.FileStates {
		if row.GameID != input.GameID {
			return errors.New("file state game ID must match input game ID")
		}
		if row.ProfileID != input.ProfileID {
			return errors.New("file state profile ID must match input profile ID")
		}
		if row.GameRelativePath == "" {
			return errors.New("file state game relative path is required")
		}
		if row.OutputKind == "" {
			return errors.New("file state output kind is required")
		}
		if row.LastAppliedAt == "" {
			return errors.New("file state last applied at is required")
		}
	}

	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}

	return *value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
