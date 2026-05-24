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
	CreatedAt          string
	UpdatedAt          string
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
