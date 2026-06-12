package dbtypes

type ModInstallConfig struct {
	ModID              int64   `db:"mod_id"`
	StrategyType       string  `db:"strategy_type"`
	TargetBase         string  `db:"target_base"`
	TargetRelativePath string  `db:"target_relative_path"`
	SourceSubpath      *string `db:"source_subpath"`
	CreatedAt          string  `db:"created_at"`
	UpdatedAt          string  `db:"updated_at"`
}

type CreateModInstallConfigInput struct {
	ModID              int64
	StrategyType       string
	TargetBase         string
	TargetRelativePath string
	SourceSubpath      *string
}

type CreateModWithInstallConfigInput struct {
	Mod     CreateModInput
	Config  CreateModInstallConfigInput
	TagIDs  []int64
	NewTags []CreateTagInput
}

type CreateModWithInstallConfigResult struct {
	Mod    Mod
	Config ModInstallConfig
	Tags   []Tag
}
