package goosemigrations

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upLegacyAppliedStateBackfill, downLegacyAppliedStateBackfill)
}

// upLegacyAppliedStateBackfill migrates legacy manifest_json rows into applied_file_states
// and applied_created_directories, then drops obsolete applied_profile_states columns.
// Temporary: remove this migration in the release after MOD-089.
func upLegacyAppliedStateBackfill(ctx context.Context, tx *sql.Tx) error {
	hasManifestColumn, err := tableHasColumn(ctx, tx, "applied_profile_states", "manifest_json")
	if err != nil {
		return err
	}
	if !hasManifestColumn {
		return nil
	}

	type appliedProfileRow struct {
		GameID       int64
		ProfileID    int64
		ManifestJSON string
		AppliedAt    string
		InstallPath  string
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			aps.game_id,
			aps.profile_id,
			aps.manifest_json,
			aps.applied_at,
			g.install_path
		FROM applied_profile_states AS aps
		INNER JOIN games AS g ON g.id = aps.game_id
		WHERE TRIM(aps.manifest_json) != ''
	`)
	if err != nil {
		return fmt.Errorf("select legacy applied profile states: %w", err)
	}
	defer rows.Close()

	var profileRows []appliedProfileRow
	for rows.Next() {
		var row appliedProfileRow
		if err := rows.Scan(&row.GameID, &row.ProfileID, &row.ManifestJSON, &row.AppliedAt, &row.InstallPath); err != nil {
			return fmt.Errorf("scan legacy applied profile state: %w", err)
		}
		profileRows = append(profileRows, row)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate legacy applied profile states: %w", err)
	}

	for _, row := range profileRows {
		hasFileStates, err := gameHasAppliedFileStates(ctx, tx, row.GameID)
		if err != nil {
			return err
		}

		document, err := decodeLegacyManifest(row.ManifestJSON)
		if err != nil {
			return fmt.Errorf("game %d: %w", row.GameID, err)
		}

		if !hasFileStates {
			fileStates, err := fileStatesFromLegacyManifest(document, row.InstallPath, row.AppliedAt)
			if err != nil {
				return fmt.Errorf("game %d: %w", row.GameID, err)
			}
			for _, state := range fileStates {
				if err := insertAppliedFileState(ctx, tx, row.GameID, row.ProfileID, row.AppliedAt, state); err != nil {
					return fmt.Errorf("game %d path %q: %w", row.GameID, state.GameRelativePath, err)
				}
			}
		}

		createdDirs, err := createdDirectoriesFromLegacyManifest(document, row.InstallPath)
		if err != nil {
			return fmt.Errorf("game %d: %w", row.GameID, err)
		}
		for _, directory := range createdDirs {
			if err := insertAppliedCreatedDirectory(ctx, tx, row.GameID, directory); err != nil {
				return fmt.Errorf("game %d directory %q: %w", row.GameID, directory.GameRelativePath, err)
			}
		}
	}

	for _, column := range []string{"manifest_json", "profile_snapshot_json", "profile_snapshot_hash"} {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			ALTER TABLE applied_profile_states DROP COLUMN %s
		`, column)); err != nil {
			return fmt.Errorf("drop column %s: %w", column, err)
		}
	}

	return nil
}

func downLegacyAppliedStateBackfill(_ context.Context, _ *sql.Tx) error {
	return nil
}

func tableHasColumn(ctx context.Context, tx *sql.Tx, tableName string, columnName string) (bool, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, fmt.Errorf("pragma table_info %s: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, fmt.Errorf("scan table_info row: %w", err)
		}
		if strings.EqualFold(name, columnName) {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func gameHasAppliedFileStates(ctx context.Context, tx *sql.Tx, gameID int64) (bool, error) {
	var count int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM applied_file_states
		WHERE game_id = ?
	`, gameID).Scan(&count); err != nil {
		return false, fmt.Errorf("count applied file states: %w", err)
	}

	return count > 0, nil
}

func insertAppliedFileState(ctx context.Context, tx *sql.Tx, gameID int64, profileID int64, appliedAt string, state legacyFileStateRow) error {
	lastAppliedAt := appliedAt
	_, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO applied_file_states (
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?)
	`, gameID, state.GameRelativePath, profileID, boolToInt(state.BaselineExists),
		nullableString(state.BaselineSHA256), nullableInt64(state.BaselineSizeBytes),
		nullableString(state.BaselineBackupPath), boolToInt(state.AppliedExists),
		nullableString(state.AppliedSHA256), nullableInt64(state.AppliedSizeBytes),
		nullableString(state.WinningSourceKind), nullableString(state.WinningSourceID),
		nullableInt64(state.WinningModID), nullableInt64(state.WinningLoadOrder),
		state.OutputKind, lastAppliedAt)
	if err != nil {
		return err
	}

	return nil
}

func insertAppliedCreatedDirectory(ctx context.Context, tx *sql.Tx, gameID int64, directory legacyCreatedDirectoryRow) error {
	_, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO applied_created_directories (
			game_id,
			game_relative_path,
			mod_id,
			mod_name
		)
		VALUES (?, ?, ?, ?)
	`, gameID, directory.GameRelativePath, nullableInt64(directory.ModID), nullableString(directory.ModName))
	if err != nil {
		return err
	}

	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return trimmed
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}

	return *value
}
