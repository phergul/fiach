package deployment

type OutputKind string

const (
	OutputCopied OutputKind = "copied"
)

type DriftKind string

const (
	DriftNone     DriftKind = "none"
	DriftMissing  DriftKind = "missing"
	DriftModified DriftKind = "modified"
	DriftExternal DriftKind = "external"
)

type FileStatus string

const (
	FileStatusAdded     FileStatus = "added"
	FileStatusReplaced  FileStatus = "replaced"
	FileStatusDeleted   FileStatus = "deleted"
	FileStatusRestored  FileStatus = "restored"
	FileStatusBlocked   FileStatus = "blocked"
	FileStatusConflict  FileStatus = "conflict"
	FileStatusDrifted   FileStatus = "drifted"
	FileStatusExternal  FileStatus = "external"
	FileStatusSkipped   FileStatus = "skipped"
	FileStatusUnchanged FileStatus = "unchanged"
)

type ConflictCategory string

const (
	ConflictExpectedOverwrite        ConflictCategory = "expected_overwrite"
	ConflictAmbiguousOverwrite       ConflictCategory = "ambiguous_overwrite"
	ConflictDestructiveFileDirectory ConflictCategory = "destructive_file_directory"
)

type RiskLevel string

const (
	RiskNone  RiskLevel = "none"
	RiskInfo  RiskLevel = "info"
	RiskError RiskLevel = "error"
)

type SourceKind string

const (
	SourceKindBaseGame SourceKind = "base_game"
	SourceKindMod      SourceKind = "mod"
)

type PlanIssueSeverity string

const (
	PlanIssueSeverityError   PlanIssueSeverity = "error"
	PlanIssueSeverityWarning PlanIssueSeverity = "warning"
)

type PlanIssueKind string

const (
	PlanIssueMissingManagedSourcePath PlanIssueKind = "missing_managed_source_path"
	PlanIssueMissingInstallConfig     PlanIssueKind = "missing_install_config"
	PlanIssueIncompleteInstallConfig  PlanIssueKind = "incomplete_install_config"
	PlanIssueMissingSourceRoot        PlanIssueKind = "missing_source_root"
	PlanIssueSourceRootNotDirectory   PlanIssueKind = "source_root_not_directory"
	PlanIssueInvalidUnrealPakSource   PlanIssueKind = "invalid_unreal_pak_source"
	PlanIssueUnsupportedStrategy      PlanIssueKind = "unsupported_strategy"
)

type ModContext struct {
	ModID   int64
	ModName string
}

type PlanIssue struct {
	Severity   PlanIssueSeverity
	Kind       PlanIssueKind
	Message    string
	ProfileID  int64
	SourcePath *string
	TargetPath *string
	Mod        *ModContext
}

type WriterEntry struct {
	Order      int
	SourceKind SourceKind
	SourceID   string
	ModID      *int64
	ModName    string
	LoadOrder  int64
	IsWinner   bool
	WouldWrite bool
}

type DesiredFile struct {
	GameRelativePath string
	SourcePath       string
	SHA256           string
	SizeBytes        int64
	OutputKind       OutputKind
	Winner           WriterEntry
	Writers          []WriterEntry
	ConflictCategory ConflictCategory
	FileStatus       FileStatus
	RiskLevel        RiskLevel
	Explanation      string
}

type DesiredState struct {
	ProfileID int64
	GameID    int64
	Files     map[string]DesiredFile
	Issues    []PlanIssue
}

func (s DesiredState) BlockingCount() int {
	count := 0
	for _, file := range s.Files {
		if file.FileStatus == FileStatusBlocked {
			count++
		}
	}
	return count
}

func (s DesiredState) CanPreview() bool {
	for _, issue := range s.Issues {
		if issue.Severity == PlanIssueSeverityError {
			return false
		}
	}
	for _, file := range s.Files {
		if file.FileStatus == FileStatusBlocked {
			return false
		}
	}
	return true
}
