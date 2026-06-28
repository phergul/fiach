package dbtypes

type AppliedCreatedDirectoryRow struct {
	GameID           int64   `db:"game_id"`
	GameRelativePath string  `db:"game_relative_path"`
	ModID            *int64  `db:"mod_id"`
	ModName          *string `db:"mod_name"`
}

type ReplaceAppliedCreatedDirectoriesInput struct {
	GameID      int64
	Directories []AppliedCreatedDirectoryRow
}
