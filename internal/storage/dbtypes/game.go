package dbtypes

type StoredGame struct {
	ID                     int64   `db:"id"`
	Name                   string  `db:"name"`
	InstallPath            string  `db:"install_path"`
	Source                 string  `db:"source"`
	SourceID               *string `db:"source_id"`
	Available              bool    `db:"available"`
	LastSeenAt             *string `db:"last_seen_at"`
	ModStoragePath         *string `db:"mod_storage_path"`
	ModStoragePathOverride *string `db:"mod_storage_path_override"`
}

type SourceGame struct {
	SourceID    string
	Name        string
	InstallPath string
}

const (
	GameSourceManual = "manual"
	GameSourceSteam  = "steam"
)

type SourceScanResult struct {
	Inserted          int
	Updated           int
	MarkedUnavailable int
	Games             []StoredGame
}
