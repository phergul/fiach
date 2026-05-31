package dto

type ModSourceType string

const (
	ModSourceTypeFolder  ModSourceType = "folder"
	ModSourceTypeArchive ModSourceType = "archive"
)

type Mod struct {
	ID                 int64
	GameID             int64
	Name               string
	SourceType         ModSourceType
	SourcePath         string
	OriginalSourcePath string
	OriginalSourceName *string
	FileCount          *int64
	DirectoryCount     *int64
	TotalSizeBytes     *int64
	MetadataJSON       *string
	Metadata           *ModMetadata
	CreatedAt          string
	UpdatedAt          string
}

type ModDeleteSummary struct {
	ModID              int64
	ModName            string
	ProfileUsageCount  int64
	IsInAppliedProfile bool
	ManagedSourcePath  string
	OriginalSourceName *string
	OriginalSourcePath string
}

type ModMetadata struct {
	ModID       int64
	Version     ModMetadataField
	Author      ModMetadataField
	Description ModMetadataField
	SourceURL   ModMetadataField
	Notes       *string
	CreatedAt   string
	UpdatedAt   string
}

type ModMetadataField struct {
	Detected  *string
	User      *string
	UserSet   bool
	Effective *string
}

type ModMetadataFieldUpdateMode string

const (
	ModMetadataFieldUpdateModeUser  ModMetadataFieldUpdateMode = "user"
	ModMetadataFieldUpdateModeClear ModMetadataFieldUpdateMode = "clear"
	ModMetadataFieldUpdateModeReset ModMetadataFieldUpdateMode = "reset"
)

type UpdateModMetadataInput struct {
	ModID       int64
	Version     ModMetadataFieldUpdate
	Author      ModMetadataFieldUpdate
	Description ModMetadataFieldUpdate
	SourceURL   ModMetadataFieldUpdate
	Notes       *string
}

type ModMetadataFieldUpdate struct {
	Mode  ModMetadataFieldUpdateMode
	Value *string
}

type ModInstallConfig struct {
	ModID              int64
	StrategyType       string
	TargetBase         string
	TargetRelativePath string
	SourceSubpath      *string
	CreatedAt          string
	UpdatedAt          string
}
