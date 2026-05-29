package dto

type StoredGame struct {
	ID                     int64
	Name                   string
	InstallPath            string
	Source                 string
	SourceID               *string
	Available              bool
	LastSeenAt             *string
	ModStoragePath         *string
	ModStoragePathOverride *string
}

type SourceScanResult struct {
	Inserted          int
	Updated           int
	MarkedUnavailable int
	Games             []StoredGame
}

type ReShadeDetectionStatus string

const (
	ReShadeDetectionStatusInstalled    ReShadeDetectionStatus = "installed"
	ReShadeDetectionStatusNotInstalled ReShadeDetectionStatus = "not_installed"
	ReShadeDetectionStatusUnsupported  ReShadeDetectionStatus = "unsupported"
)

type ReShadeTarget struct {
	Path        string
	Executables []string
}

type ReShadeDetectionResult struct {
	Status            ReShadeDetectionStatus
	Targets           []ReShadeTarget
	UnsupportedReason *string
}
