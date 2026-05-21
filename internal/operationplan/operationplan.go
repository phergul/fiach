package operationplan

type OperationType string

const (
	OperationTypeCopy            OperationType = "copy"
	OperationTypeReplace         OperationType = "replace"
	OperationTypeCreateDirectory OperationType = "create_directory"
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
	PlanIssueMissingGameInstallPath   PlanIssueKind = "missing_game_install_path"
	PlanIssueMissingGameInstallDir    PlanIssueKind = "missing_game_install_directory"
	PlanIssueGameInstallPathNotDir    PlanIssueKind = "game_install_path_not_directory"
	PlanIssueMissingSourceRoot        PlanIssueKind = "missing_source_root"
	PlanIssueSourceRootNotDirectory   PlanIssueKind = "source_root_not_directory"
	PlanIssueMissingGameModStorage    PlanIssueKind = "missing_game_mod_storage_path"
	PlanIssueTargetDirectoryPathFile  PlanIssueKind = "target_directory_path_is_file"
	PlanIssueTargetFilePathDirectory  PlanIssueKind = "target_file_path_is_directory"
	PlanIssueTargetPathConflict       PlanIssueKind = "target_path_conflict"
	PlanIssueReplaceExistingTarget    PlanIssueKind = "replace_existing_target"
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

type Operation struct {
	Type       OperationType
	SourcePath *string
	TargetPath string
	BackupPath *string
	Conflict   bool
	Mod        ModContext
}

type OperationPlan struct {
	Operations []Operation
	Issues     []PlanIssue
	CanApply   bool
}

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
}
