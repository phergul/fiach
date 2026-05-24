package dto

type ApplyOperationStatus string

const (
	ApplyOperationStatusCompleted ApplyOperationStatus = "completed"
	ApplyOperationStatusFailed    ApplyOperationStatus = "failed"
	ApplyOperationStatusSkipped   ApplyOperationStatus = "skipped"
)

type ApplyOperationResult struct {
	OperationIndex int
	Operation      Operation
	Status         ApplyOperationStatus
	Message        string
	Error          *string
}

type ApplyOperationPlanResult struct {
	Success        bool
	CompletedCount int
	FailedCount    int
	SkippedCount   int
	Results        []ApplyOperationResult
	Manifest       AppliedOperationManifest
}

type AppliedOperationManifest struct {
	AddedFiles         []AppliedFileManifestEntry
	ReplacedFiles      []ReplacedFileManifestEntry
	CreatedDirectories []AppliedDirectoryManifestEntry
}

type AppliedFileManifestEntry struct {
	OperationIndex int
	Mod            ModContext
	SourcePath     string
	TargetPath     string
	SHA256         string
	SizeBytes      int64
}

type ReplacedFileManifestEntry struct {
	OperationIndex  int
	Mod             ModContext
	SourcePath      string
	TargetPath      string
	SHA256          string
	SizeBytes       int64
	BackupPath      string
	BackupSHA256    string
	BackupSizeBytes int64
}

type AppliedDirectoryManifestEntry struct {
	OperationIndex int
	Mod            ModContext
	TargetPath     string
}
