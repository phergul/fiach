package restoreplan

import "github.com/phergul/mod-manager/internal/fileops"

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

type RestoreOperation struct {
	Type                   RestoreOperationType
	ManifestOperationIndex int
	Mod                    Mod
	TargetPath             string
	BackupPath             *string
}

type Mod struct {
	ID   int64
	Name string
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

type Context struct {
	GameInstallPath    string
	GameModStoragePath string
}

type resolvedContext struct {
	gameInstallPath    string
	gameModStoragePath string
}

var computeFileIntegrity = fileops.FileIntegrity
