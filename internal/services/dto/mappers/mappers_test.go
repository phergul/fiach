package mappers

import (
	"testing"

	"github.com/phergul/fiach/internal/operationplan"
	"github.com/phergul/fiach/internal/restoreplan"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestOperationPlanDTORoundTrip(t *testing.T) {
	t.Parallel()

	sourcePath := "/managed/mod/file.txt"
	backupPath := "/managed/storage/backup/file.txt"
	targetPath := "/games/game/file.txt"
	plan := operationplan.OperationPlan{
		CanApply: true,
		Operations: []operationplan.Operation{
			{
				Type:       operationplan.OperationTypeReplace,
				SourcePath: &sourcePath,
				TargetPath: targetPath,
				BackupPath: &backupPath,
				Conflict:   true,
				Mod:        operationplan.ModContext{ModID: 7, ModName: "SkyUI"},
			},
		},
		Issues: []operationplan.PlanIssue{
			{
				Severity:                    operationplan.PlanIssueSeverityWarning,
				Kind:                        operationplan.PlanIssueReplaceExistingTarget,
				Message:                     "Target exists.",
				ProfileID:                   3,
				SourcePath:                  &sourcePath,
				TargetPath:                  &targetPath,
				Mod:                         &operationplan.ModContext{ModID: 7, ModName: "SkyUI"},
				ConflictingOperationIndexes: []int{0, 2},
			},
		},
	}

	roundTrip := ToInternalOperationPlan(ToDTOOperationPlan(plan))
	if roundTrip.Operations[0].Type != plan.Operations[0].Type ||
		*roundTrip.Operations[0].SourcePath != sourcePath ||
		*roundTrip.Operations[0].BackupPath != backupPath ||
		roundTrip.Issues[0].Kind != plan.Issues[0].Kind ||
		roundTrip.Issues[0].Mod.ModID != 7 ||
		len(roundTrip.Issues[0].ConflictingOperationIndexes) != 2 ||
		roundTrip.Issues[0].ConflictingOperationIndexes[0] != 0 ||
		roundTrip.Issues[0].ConflictingOperationIndexes[1] != 2 {
		t.Fatalf("operation plan round trip = %+v, want preserved plan", roundTrip)
	}
}

func TestApplyResultDTOIncludesManifest(t *testing.T) {
	t.Parallel()

	result := operationplan.ApplyOperationPlanResult{
		Success:        true,
		CompletedCount: 1,
		Results: []operationplan.ApplyOperationResult{
			{
				OperationIndex: 0,
				Status:         operationplan.ApplyOperationStatusCompleted,
				Message:        "Copied file.",
				Operation: operationplan.Operation{
					Type:       operationplan.OperationTypeCopy,
					TargetPath: "/games/game/Data/file.txt",
					Mod:        operationplan.ModContext{ModID: 9, ModName: "Patch"},
				},
			},
		},
		Manifest: operationplan.AppliedOperationManifest{
			AddedFiles: []operationplan.AppliedFileManifestEntry{
				{
					OperationIndex: 0,
					Mod:            operationplan.ModContext{ModID: 9, ModName: "Patch"},
					SourcePath:     "/managed/patch/file.txt",
					TargetPath:     "/games/game/Data/file.txt",
					SHA256:         "abc",
					SizeBytes:      12,
				},
			},
		},
	}

	dtoResult := ToDTOApplyOperationPlanResult(result)
	if dtoResult.Results[0].Status != dto.ApplyOperationStatusCompleted ||
		dtoResult.Manifest.AddedFiles[0].Mod.ModID != 9 ||
		dtoResult.Manifest.AddedFiles[0].SHA256 != "abc" {
		t.Fatalf("ToDTOApplyOperationPlanResult() = %+v, want result and manifest fields", dtoResult)
	}
}

func TestRestoreResultDTOConversion(t *testing.T) {
	t.Parallel()

	message := "backup missing"
	result := restoreplan.RestoreResult{
		Success:     false,
		FailedCount: 1,
		Results: []restoreplan.RestoreOperationResult{
			{
				OperationIndex: 2,
				Status:         restoreplan.RestoreOperationStatusFailed,
				Message:        "Failed.",
				Error:          &message,
				Operation: restoreplan.RestoreOperation{
					Type:                   restoreplan.RestoreOperationTypeRestoreReplacedFile,
					ManifestOperationIndex: 5,
					Mod:                    restoreplan.Mod{ID: 4, Name: "Textures"},
					TargetPath:             "/games/game/Data/texture.dds",
				},
			},
		},
	}

	dtoResult := ToDTORestoreResult(result)
	if dtoResult.Results[0].Status != dto.RestoreOperationStatusFailed ||
		dtoResult.Results[0].Operation.Mod.ID != 4 ||
		*dtoResult.Results[0].Error != message {
		t.Fatalf("ToDTORestoreResult() = %+v, want restore operation fields", dtoResult)
	}
}

func TestStorageDTOConversionPreservesNullableFields(t *testing.T) {
	t.Parallel()

	sourceID := "10"
	lastSeenAt := "2026-05-24T12:00:00Z"
	overridePath := "/custom/mods"
	game := dbtypes.StoredGame{
		ID:                     1,
		Name:                   "Skyrim",
		SourceID:               &sourceID,
		LastSeenAt:             &lastSeenAt,
		ModStoragePathOverride: &overridePath,
	}

	dtoGame := ToDTOStoredGame(game)
	if dtoGame.SourceID == nil || *dtoGame.SourceID != sourceID ||
		dtoGame.LastSeenAt == nil || *dtoGame.LastSeenAt != lastSeenAt ||
		dtoGame.ModStoragePathOverride == nil || *dtoGame.ModStoragePathOverride != overridePath {
		t.Fatalf("ToDTOStoredGame() = %+v, want nullable pointers preserved", dtoGame)
	}
}
