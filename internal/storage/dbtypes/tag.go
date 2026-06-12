package dbtypes

type TagColor string

const (
	TagColorRed    TagColor = "red"
	TagColorOrange TagColor = "orange"
	TagColorYellow TagColor = "yellow"
	TagColorGreen  TagColor = "green"
	TagColorTeal   TagColor = "teal"
	TagColorBlue   TagColor = "blue"
	TagColorPurple TagColor = "purple"
	TagColorPink   TagColor = "pink"
)

type Tag struct {
	ID             int64    `db:"id"`
	GameID         int64    `db:"game_id"`
	Name           string   `db:"name"`
	NormalizedName string   `db:"normalized_name"`
	Color          TagColor `db:"color"`
	CreatedAt      string   `db:"created_at"`
	UpdatedAt      string   `db:"updated_at"`
}

type CreateTagInput struct {
	Name  string
	Color TagColor
}

type SetModTagsInput struct {
	ModID     int64
	TagIDs    []int64
	NewTags   []CreateTagInput
	MergeOnly bool
}

type UpdateModDetailsInput struct {
	ModID    int64
	Name     string
	Metadata UpdateModMetadataInput
	TagIDs   []int64
	NewTags  []CreateTagInput
}
