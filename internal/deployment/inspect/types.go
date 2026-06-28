package inspect

type StateKind string

const (
	StateBaseline StateKind = "baseline"
	StateApplied  StateKind = "applied"
	StateCurrent  StateKind = "current"
	StateDesired  StateKind = "desired"
)

type InspectionKind string

const (
	InspectionTextDiff       InspectionKind = "text_diff"
	InspectionPEMetadata     InspectionKind = "pe_metadata"
	InspectionImageMetadata  InspectionKind = "image_metadata"
	InspectionArchiveListing InspectionKind = "archive_listing"
	InspectionBinaryFallback InspectionKind = "binary_fallback"
)

type ComparePair struct {
	Left  StateKind
	Right StateKind
}

type SideMetadata struct {
	StateKind         StateKind
	Label             string
	Available         bool
	UnavailableReason string
	SHA256            string
	SizeBytes         int64
}

type TextDiffLine struct {
	Kind   string
	Line   string
	LineNo int
}

type PEMetadata struct {
	Machine         string
	SectionCount    int
	Characteristics string
	IsDLL           bool
	IsEXE           bool
	SHA256          string
	SizeBytes       int64
}

type ImageMetadata struct {
	Format    string
	Width     int
	Height    int
	SHA256    string
	SizeBytes int64
}

type ArchiveEntry struct {
	Path        string
	SizeBytes   int64
	IsDirectory bool
}

type InspectionResult struct {
	RelativePath        string
	Kind                InspectionKind
	LeftState           StateKind
	RightState          StateKind
	Left                SideMetadata
	Right               SideMetadata
	TextLines           []TextDiffLine
	PEMetadataLeft      *PEMetadata
	PEMetadataRight     *PEMetadata
	ImageMetadataLeft   *ImageMetadata
	ImageMetadataRight  *ImageMetadata
	ArchiveEntriesLeft  []ArchiveEntry
	ArchiveEntriesRight []ArchiveEntry
	LimitReached        bool
	LimitReason         string
	FallbackReason      string
}
