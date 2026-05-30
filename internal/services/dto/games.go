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
