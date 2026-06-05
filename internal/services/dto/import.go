package dto

type PreValidateImportInput struct {
	SourceType ModSourceType
	SourcePath string
}

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

type UpdateModInput struct {
	ModID      int64
	SourceType ModSourceType
	SourcePath string
}

type UpdateModResult struct {
	Mod                Mod
	Before             ModPackageSnapshot
	After              ModPackageSnapshot
	MetadataWarning    *string
	IsInAppliedProfile bool
	RequiresReapply    bool
}

type ModPackageSnapshot struct {
	SourceType         ModSourceType
	OriginalSourcePath string
	OriginalSourceName *string
	FileCount          *int64
	DirectoryCount     *int64
	TotalSizeBytes     *int64
	DetectedMetadata   ModDetectedMetadataSnapshot
}

type ModDetectedMetadataSnapshot struct {
	Version     *string
	Author      *string
	Description *string
	SourceURL   *string
}
