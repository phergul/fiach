package inspect

const (
	MaxTextDiffBytes          int64 = 1 << 20
	MaxArchiveFiles           int   = 500
	MaxArchivePathDepth       int   = 2
	MaxArchiveUncompressedSum int64 = 100 << 20
	TextSniffBytes            int   = 8192
)
