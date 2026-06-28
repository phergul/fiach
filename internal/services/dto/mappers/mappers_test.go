package mappers

import (
	"testing"

	"github.com/phergul/fiach/internal/restoreplan"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

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
