package dto

type PreValidateImportInput struct {
	SourceType ModSourceType
	SourcePath string
}

type PreValidateImportResult struct {
	SuggestedStrategy *StrategyType
}

type ImportTargetDetectionResult struct {
	Candidates []string
	Warnings   []string
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
	TagIDs             []int64
	NewTags            []CreateTagInput
}

type ImportModResult struct {
	Mod      Mod
	Config   ModInstallConfig
	Warnings []string
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
	Warnings           []string
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

type ImportSourceRef struct {
	SourceType ModSourceType
	SourcePath string
}

type ResolveImportSourceDuplicatesInput struct {
	GameID  int64
	Sources []ImportSourceRef
}

type ImportSourceDuplicateStatus struct {
	SourceType      ModSourceType
	SourcePath      string
	CanonicalPath   string
	Error           *string
	IsDuplicate     bool
	ExistingModID   *int64
	ExistingModName *string
}

type ResolveImportSourceDuplicatesResult struct {
	Items []ImportSourceDuplicateStatus
}
