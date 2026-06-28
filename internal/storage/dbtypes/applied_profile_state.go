package dbtypes

type AppliedProfileState struct {
	GameID                         int64   `db:"game_id"`
	ProfileID                      int64   `db:"profile_id"`
	ProfileCompositionSnapshotJSON *string `db:"profile_composition_snapshot_json"`
	ProfileCompositionSnapshotHash *string `db:"profile_composition_snapshot_hash"`
	AppliedAt                      string  `db:"applied_at"`
}

type SaveAppliedProfileStateInput struct {
	GameID                         int64
	ProfileID                      int64
	ProfileCompositionSnapshotJSON *string
	ProfileCompositionSnapshotHash *string
	FileStates                     []AppliedFileStateRow
	ReplaceFileStates              bool
	CreatedDirectories             []AppliedCreatedDirectoryRow
	ReplaceCreatedDirectories      bool
}
