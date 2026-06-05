package dto

type ModProfile struct {
	ID        int64
	GameID    int64
	Name      string
	CreatedAt string
	UpdatedAt string
}

type ProfileMod struct {
	ProfileID    int64
	ModID        int64
	Name         string
	SourcePath   string
	ModUpdatedAt string
	Enabled      bool
	LoadOrder    int64
	CreatedAt    string
	UpdatedAt    string
}

type AppliedProfileSummary struct {
	GameID                   int64
	ProfileID                int64
	ProfileName              string
	AppliedAt                string
	HasAppliedProfileChanged *bool
}
