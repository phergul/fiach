package dbtypes

type AppliedProfileState struct {
	GameID                         int64   `db:"game_id"`
	ProfileID                      int64   `db:"profile_id"`
	ManifestJSON                   string  `db:"manifest_json"`
	ProfileSnapshotJSON            string  `db:"profile_snapshot_json"`
	ProfileSnapshotHash            string  `db:"profile_snapshot_hash"`
	ProfileCompositionSnapshotJSON *string `db:"profile_composition_snapshot_json"`
	ProfileCompositionSnapshotHash *string `db:"profile_composition_snapshot_hash"`
	AppliedAt                      string  `db:"applied_at"`
}

type SaveAppliedProfileStateInput struct {
	GameID                         int64
	ProfileID                      int64
	ManifestJSON                   string
	ProfileSnapshotJSON            string
	ProfileSnapshotHash            string
	ProfileCompositionSnapshotJSON *string
	ProfileCompositionSnapshotHash *string
	FileStates                     []AppliedFileStateRow
	ReplaceFileStates              bool
}
