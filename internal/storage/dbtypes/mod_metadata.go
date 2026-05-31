package dbtypes

type ModMetadata struct {
	ModID               int64   `db:"mod_id"`
	DetectedVersion     *string `db:"detected_version"`
	UserVersion         *string `db:"user_version"`
	VersionUserSet      bool    `db:"version_user_set"`
	DetectedAuthor      *string `db:"detected_author"`
	UserAuthor          *string `db:"user_author"`
	AuthorUserSet       bool    `db:"author_user_set"`
	DetectedDescription *string `db:"detected_description"`
	UserDescription     *string `db:"user_description"`
	DescriptionUserSet  bool    `db:"description_user_set"`
	DetectedSourceURL   *string `db:"detected_source_url"`
	UserSourceURL       *string `db:"user_source_url"`
	SourceURLUserSet    bool    `db:"source_url_user_set"`
	Notes               *string `db:"notes"`
	CreatedAt           string  `db:"created_at"`
	UpdatedAt           string  `db:"updated_at"`
}

type ModMetadataDetectedInput struct {
	Version     *string
	Author      *string
	Description *string
	SourceURL   *string
}

type UpdateModMetadataInput struct {
	ModID       int64
	Version     ModMetadataFieldUpdate
	Author      ModMetadataFieldUpdate
	Description ModMetadataFieldUpdate
	SourceURL   ModMetadataFieldUpdate
	Notes       *string
}

type ModMetadataFieldUpdate struct {
	UserSet bool
	Value   *string
}
