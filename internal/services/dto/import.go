package dto

type PreviewImportConfigurationInput struct {
	GameID             int64
	SourceType         ModSourceType
	SourcePath         string
	StrategyType       StrategyType
	TargetRelativePath string
}

type ImportModInput struct {
	GameID             int64
	Name               string
	SourceType         ModSourceType
	SourcePath         string
	StrategyType       StrategyType
	TargetRelativePath string
}

type ImportModResult struct {
	Mod    Mod
	Config ModInstallConfig
}
