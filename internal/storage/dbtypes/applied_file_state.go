package dbtypes

type AppliedFileStateRow struct {
	GameID             int64   `db:"game_id"`
	GameRelativePath   string  `db:"game_relative_path"`
	ProfileID          int64   `db:"profile_id"`
	BaselineExists     bool    `db:"baseline_exists"`
	BaselineSHA256     *string `db:"baseline_sha256"`
	BaselineSizeBytes  *int64  `db:"baseline_size_bytes"`
	BaselineBackupPath *string `db:"baseline_backup_path"`
	AppliedExists      bool    `db:"applied_exists"`
	AppliedSHA256      *string `db:"applied_sha256"`
	AppliedSizeBytes   *int64  `db:"applied_size_bytes"`
	WinningSourceKind  *string `db:"winning_source_kind"`
	WinningSourceID    *string `db:"winning_source_id"`
	WinningModID       *int64  `db:"winning_mod_id"`
	WinningLoadOrder   *int64  `db:"winning_load_order"`
	OutputKind         string  `db:"output_kind"`
	UserDecision       *string `db:"user_decision"`
	LastAppliedAt      string  `db:"last_applied_at"`
}

type ReplaceAppliedFileStatesInput struct {
	GameID     int64
	ProfileID  int64
	FileStates []AppliedFileStateRow
}
