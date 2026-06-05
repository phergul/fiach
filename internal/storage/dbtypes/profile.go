package dbtypes

type ModProfile struct {
	ID        int64  `db:"id"`
	GameID    int64  `db:"game_id"`
	Name      string `db:"name"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
}

type ProfileMod struct {
	ProfileID    int64  `db:"profile_id"`
	ModID        int64  `db:"mod_id"`
	Name         string `db:"name"`
	SourcePath   string `db:"source_path"`
	ModUpdatedAt string `db:"mod_updated_at"`
	Enabled      bool   `db:"enabled"`
	LoadOrder    int64  `db:"load_order"`
	CreatedAt    string `db:"created_at"`
	UpdatedAt    string `db:"updated_at"`
}
