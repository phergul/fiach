package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func (s *Store) SaveAppliedProfileState(ctx context.Context, input dbtypes.SaveAppliedProfileStateInput) (state dbtypes.AppliedProfileState, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("upsert applied profile state row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.AppliedProfileState{}, errors.New("store is not open")
	}
	if err := validateSaveAppliedProfileStateInput(input); err != nil {
		return dbtypes.AppliedProfileState{}, err
	}

	err = withTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		if err := requireProfileBelongsToGame(ctx, tx, input.ProfileID, input.GameID); err != nil {
			return err
		}

		appliedAt := FormatAppliedTimestamp(time.Now().UTC())

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO applied_profile_states (
				game_id,
				profile_id,
				profile_composition_snapshot_json,
				profile_composition_snapshot_hash,
				applied_at
			)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(game_id) DO UPDATE SET
				profile_id = excluded.profile_id,
				profile_composition_snapshot_json = excluded.profile_composition_snapshot_json,
				profile_composition_snapshot_hash = excluded.profile_composition_snapshot_hash,
				applied_at = excluded.applied_at
		`, input.GameID, input.ProfileID,
			nullableText(cleanOptionalString(input.ProfileCompositionSnapshotJSON)),
			nullableText(cleanOptionalString(input.ProfileCompositionSnapshotHash)),
			appliedAt); err != nil {
			return err
		}

		var found bool
		var err error
		state, found, err = getAppliedProfileState(ctx, tx, input.GameID)
		if err != nil {
			return err
		}
		if !found {
			return sql.ErrNoRows
		}

		if input.ReplaceFileStates || len(input.FileStates) > 0 {
			fileStates := make([]dbtypes.AppliedFileStateRow, len(input.FileStates))
			copy(fileStates, input.FileStates)
			for index := range fileStates {
				fileStates[index].LastAppliedAt = state.AppliedAt
			}
			if err := replaceAppliedFileStates(ctx, tx, dbtypes.ReplaceAppliedFileStatesInput{
				GameID:     input.GameID,
				ProfileID:  input.ProfileID,
				FileStates: fileStates,
			}); err != nil {
				return err
			}
		}

		if input.ReplaceCreatedDirectories || len(input.CreatedDirectories) > 0 {
			directories := make([]dbtypes.AppliedCreatedDirectoryRow, len(input.CreatedDirectories))
			copy(directories, input.CreatedDirectories)
			for index := range directories {
				directories[index].GameID = input.GameID
			}
			if err := replaceAppliedCreatedDirectories(ctx, tx, dbtypes.ReplaceAppliedCreatedDirectoriesInput{
				GameID:      input.GameID,
				Directories: directories,
			}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return dbtypes.AppliedProfileState{}, err
	}

	return state, nil
}

func (s *Store) GetAppliedProfileState(ctx context.Context, gameID int64) (state dbtypes.AppliedProfileState, found bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select applied profile state: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.AppliedProfileState{}, false, errors.New("store is not open")
	}
	if gameID <= 0 {
		return dbtypes.AppliedProfileState{}, false, errors.New("game ID must be positive")
	}

	state, found, err = getAppliedProfileState(ctx, s.db, gameID)
	if err != nil {
		return dbtypes.AppliedProfileState{}, false, err
	}

	return state, found, nil
}

func (s *Store) DeleteAppliedProfileState(ctx context.Context, gameID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("delete applied profile state row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}
	if gameID <= 0 {
		return errors.New("game ID must be positive")
	}

	_, err = s.db.ExecContext(ctx, `
		DELETE FROM applied_profile_states
		WHERE game_id = ?
	`, gameID)
	if err != nil {
		return err
	}

	return nil
}

func validateSaveAppliedProfileStateInput(input dbtypes.SaveAppliedProfileStateInput) error {
	if input.GameID <= 0 {
		return errors.New("game ID must be positive")
	}
	if input.ProfileID <= 0 {
		return errors.New("profile ID must be positive")
	}
	if (input.ProfileCompositionSnapshotJSON == nil) != (input.ProfileCompositionSnapshotHash == nil) {
		return errors.New("profile composition snapshot JSON and hash must be provided together")
	}
	if input.ProfileCompositionSnapshotJSON != nil {
		compositionJSON := strings.TrimSpace(*input.ProfileCompositionSnapshotJSON)
		if compositionJSON == "" {
			return errors.New("profile composition snapshot JSON is required when provided")
		}
		if !json.Valid([]byte(compositionJSON)) {
			return errors.New("profile composition snapshot JSON is invalid")
		}
	}
	if input.ProfileCompositionSnapshotHash != nil {
		if strings.TrimSpace(*input.ProfileCompositionSnapshotHash) == "" {
			return errors.New("profile composition snapshot hash is required when provided")
		}
	}

	return nil
}

func requireProfileBelongsToGame(ctx context.Context, db appliedProfileStateGetter, profileID int64, gameID int64) error {
	var count int
	if err := db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM profiles
		WHERE id = ?
			AND game_id = ?
	`, profileID, gameID); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("profile %d does not belong to game %d", profileID, gameID)
	}

	return nil
}

type appliedProfileStateGetter interface {
	GetContext(context.Context, any, string, ...any) error
}

func getAppliedProfileState(ctx context.Context, db appliedProfileStateGetter, gameID int64) (dbtypes.AppliedProfileState, bool, error) {
	var state dbtypes.AppliedProfileState
	err := db.GetContext(ctx, &state, `
		SELECT
			game_id,
			profile_id,
			profile_composition_snapshot_json,
			profile_composition_snapshot_hash,
			applied_at
		FROM applied_profile_states
		WHERE game_id = ?
	`, gameID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dbtypes.AppliedProfileState{}, false, nil
		}
		return dbtypes.AppliedProfileState{}, false, err
	}

	return state, true, nil
}
