package execute

import (
	"github.com/phergul/fiach/internal/deployment/planner"
)

type VanillaRestoreOperationType string

const (
	VanillaRestoreOperationRemoveAddedFile      VanillaRestoreOperationType = "remove_added_file"
	VanillaRestoreOperationRestoreReplacedFile  VanillaRestoreOperationType = "restore_replaced_file"
	VanillaRestoreOperationRemoveCreatedDir     VanillaRestoreOperationType = "remove_created_directory"
	VanillaRestoreOperationDeleteRestoredBackup VanillaRestoreOperationType = "delete_restored_backup"
)

type VanillaRestoreOperationStatus string

const (
	VanillaRestoreOperationStatusCompleted VanillaRestoreOperationStatus = "completed"
	VanillaRestoreOperationStatusFailed    VanillaRestoreOperationStatus = "failed"
	VanillaRestoreOperationStatusSkipped   VanillaRestoreOperationStatus = "skipped"
)

type VanillaRestoreMod struct {
	ID   int64
	Name string
}

type VanillaRestoreOperation struct {
	Type                   VanillaRestoreOperationType
	ManifestOperationIndex int
	Mod                    VanillaRestoreMod
	TargetPath             string
	BackupPath             *string
}

type VanillaRestoreOperationResult struct {
	OperationIndex int
	Operation      VanillaRestoreOperation
	Status         VanillaRestoreOperationStatus
	Message        string
	Error          *string
}

type VanillaRestoreResult struct {
	Success        bool
	CompletedCount int
	FailedCount    int
	SkippedCount   int
	Results        []VanillaRestoreOperationResult
}

type RestoreContext struct {
	GameID             int64
	GameInstallPath    string
	GameModStoragePath string
	Plan               planner.DeploymentPlan
	CreatedDirectories []RestoreCreatedDirectory
}

type RestoreCreatedDirectory struct {
	GameRelativePath string
	ModID            *int64
	ModName          *string
}
