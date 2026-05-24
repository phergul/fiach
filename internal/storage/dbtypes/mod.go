package dbtypes

type Mod struct {
	ID                 int64         `db:"id"`
	GameID             int64         `db:"game_id"`
	Name               string        `db:"name"`
	SourceType         ModSourceType `db:"source_type"`
	SourcePath         string        `db:"source_path"`
	OriginalSourcePath string        `db:"original_source_path"`
	OriginalSourceName *string       `db:"original_source_name"`
	CreatedAt          string        `db:"created_at"`
	UpdatedAt          string        `db:"updated_at"`
}

type ModSourceType string

const (
	ModSourceTypeFolder  ModSourceType = "folder"
	ModSourceTypeArchive ModSourceType = "archive"
)

type CreateModInput struct {
	GameID             int64
	Name               string
	SourceType         ModSourceType
	SourcePath         string
	OriginalSourcePath string
	OriginalSourceName *string
}
