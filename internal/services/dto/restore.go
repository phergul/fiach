package dto

type RestoreOperationType string

const (
	RestoreOperationTypeRemoveAddedFile      RestoreOperationType = "remove_added_file"
	RestoreOperationTypeRestoreReplacedFile  RestoreOperationType = "restore_replaced_file"
	RestoreOperationTypeRemoveCreatedDir     RestoreOperationType = "remove_created_directory"
	RestoreOperationTypeDeleteRestoredBackup RestoreOperationType = "delete_restored_backup"
)

type RestoreOperationStatus string

const (
	RestoreOperationStatusCompleted RestoreOperationStatus = "completed"
	RestoreOperationStatusFailed    RestoreOperationStatus = "failed"
	RestoreOperationStatusSkipped   RestoreOperationStatus = "skipped"
)

type RestoreMod struct {
	ID   int64
	Name string
}

type RestoreOperation struct {
	Type                   RestoreOperationType
	ManifestOperationIndex int
	Mod                    RestoreMod
	TargetPath             string
	BackupPath             *string
}

type RestoreOperationResult struct {
	OperationIndex int
	Operation      RestoreOperation
	Status         RestoreOperationStatus
	Message        string
	Error          *string
}

type RestoreResult struct {
	Success        bool
	CompletedCount int
	FailedCount    int
	SkippedCount   int
	Results        []RestoreOperationResult
}
