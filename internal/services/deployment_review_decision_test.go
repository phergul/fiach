package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/deployment/drift"
	"github.com/phergul/fiach/internal/installconfig"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestDeploymentReviewServiceSetDriftDecisionRefreshesPreviewAndUnblocksApply(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	sourcePath := makeProfilePlanSourceTree(t, map[string]string{
		"Data/SkyUI.esp": "plugin",
	})
	modID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", sourcePath)

	addServiceProfileMod(t, store, profileID, modID, true, 0)
	addServiceInstallConfig(t, store, modID, string(dto.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Mods/SkyUI", nil)

	targetPath := filepath.Join(gameRoot, "Mods", "SkyUI", "Data", "SkyUI.esp")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("external-edit"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appliedSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
	appliedSize := int64(1)
	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profileID,
		ManifestJSON:        `{"version":2,"createdDirectories":[],"addedFiles":[],"replacedFiles":[],"files":{"Mods/SkyUI/Data/SkyUI.esp":{"gameRelativePath":"Mods/SkyUI/Data/SkyUI.esp","outputKind":"copied","appliedExists":true,"appliedSHA256":"0000000000000000000000000000000000000000000000000000000000000000","appliedSizeBytes":1}}}`,
		ProfileSnapshotJSON: `{"version":2}`,
		ProfileSnapshotHash: "snapshot",
		FileStates: []dbtypes.AppliedFileStateRow{
			{
				GameID:           gameID,
				GameRelativePath: "Mods/SkyUI/Data/SkyUI.esp",
				ProfileID:        profileID,
				AppliedExists:    true,
				AppliedSHA256:    &appliedSHA256,
				AppliedSizeBytes: &appliedSize,
				OutputKind:       "copied",
				LastAppliedAt:    "2026-06-27T00:00:00Z",
			},
		},
	}); err != nil {
		t.Fatalf("SaveAppliedProfileState() error = %v", err)
	}

	service := newDeploymentReviewTestService(store)
	preview, err := service.BuildDeploymentReviewPreview(context.Background(), profileID)
	if err != nil {
		t.Fatalf("BuildDeploymentReviewPreview() error = %v", err)
	}
	if preview.Summary.CanApply {
		t.Fatal("initial preview CanApply = true, want false")
	}

	detail, err := service.GetDeploymentFileDetail(context.Background(), preview.PreviewHash, "Mods/SkyUI/Data/SkyUI.esp")
	if err != nil {
		t.Fatalf("GetDeploymentFileDetail() error = %v", err)
	}
	if len(detail.AvailableActions) != 3 {
		t.Fatalf("AvailableActions = %+v, want three drift actions", detail.AvailableActions)
	}

	updatedPreview, err := service.SetDeploymentDriftDecision(
		context.Background(),
		profileID,
		preview.PreviewHash,
		"Mods/SkyUI/Data/SkyUI.esp",
		drift.UserDecisionKeepExternal,
	)
	if err != nil {
		t.Fatalf("SetDeploymentDriftDecision() error = %v", err)
	}
	if updatedPreview.PreviewHash == preview.PreviewHash {
		t.Fatal("updated preview hash = original hash, want refreshed hash")
	}
	if !updatedPreview.Summary.CanApply {
		t.Fatal("updated preview CanApply = false, want true after keep_external")
	}
	if updatedPreview.Summary.StatusCounts["external"] != 1 {
		t.Fatalf("status counts = %+v, want one external path", updatedPreview.Summary.StatusCounts)
	}

	updatedDetail, err := service.GetDeploymentFileDetail(
		context.Background(),
		updatedPreview.PreviewHash,
		"Mods/SkyUI/Data/SkyUI.esp",
	)
	if err != nil {
		t.Fatalf("GetDeploymentFileDetail(updated) error = %v", err)
	}
	if updatedDetail.UserDecision != drift.UserDecisionKeepExternal {
		t.Fatalf("UserDecision = %q, want keep_external", updatedDetail.UserDecision)
	}
	if len(updatedDetail.AvailableActions) != 1 || updatedDetail.AvailableActions[0] != drift.UserDecisionClear {
		t.Fatalf("AvailableActions = %+v, want clear", updatedDetail.AvailableActions)
	}

	rebuiltPreview, err := service.BuildDeploymentReviewPreview(context.Background(), profileID)
	if err != nil {
		t.Fatalf("BuildDeploymentReviewPreview(rebuild) error = %v", err)
	}
	if rebuiltPreview.Summary.StatusCounts["external"] != 1 {
		t.Fatalf("rebuilt status counts = %+v, want persisted external decision", rebuiltPreview.Summary.StatusCounts)
	}
}

func TestDeploymentReviewServiceSetDriftDecisionRejectsStalePreview(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", t.TempDir())
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	service := newDeploymentReviewTestService(store)

	_, err := service.SetDeploymentDriftDecision(
		context.Background(),
		profileID,
		"stale-hash",
		"Data/file.esp",
		drift.UserDecisionSkipped,
	)
	if err == nil {
		t.Fatal("SetDeploymentDriftDecision() error = nil, want stale preview rejection")
	}
	if apperror.UserMessage(err) != "The deployment preview is no longer available. Refresh the preview and try again." {
		t.Fatalf("SetDeploymentDriftDecision() error = %q, want stale preview rejection", apperror.UserMessage(err))
	}
}

func TestDeploymentReviewServiceSetDriftDecisionRejectsInvalidDecision(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	sourcePath := makeProfilePlanSourceTree(t, map[string]string{
		"Data/SkyUI.esp": "plugin",
	})
	modID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", sourcePath)

	addServiceProfileMod(t, store, profileID, modID, true, 0)
	addServiceInstallConfig(t, store, modID, string(dto.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Mods/SkyUI", nil)

	targetPath := filepath.Join(gameRoot, "Mods", "SkyUI", "Data", "SkyUI.esp")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("external-edit"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appliedSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
	appliedSize := int64(1)
	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profileID,
		ManifestJSON:        `{"version":2,"createdDirectories":[],"addedFiles":[],"replacedFiles":[],"files":{"Mods/SkyUI/Data/SkyUI.esp":{"gameRelativePath":"Mods/SkyUI/Data/SkyUI.esp","outputKind":"copied","appliedExists":true,"appliedSHA256":"0000000000000000000000000000000000000000000000000000000000000000","appliedSizeBytes":1}}}`,
		ProfileSnapshotJSON: `{"version":2}`,
		ProfileSnapshotHash: "snapshot",
		FileStates: []dbtypes.AppliedFileStateRow{
			{
				GameID:           gameID,
				GameRelativePath: "Mods/SkyUI/Data/SkyUI.esp",
				ProfileID:        profileID,
				AppliedExists:    true,
				AppliedSHA256:    &appliedSHA256,
				AppliedSizeBytes: &appliedSize,
				OutputKind:       "copied",
				LastAppliedAt:    "2026-06-27T00:00:00Z",
			},
		},
	}); err != nil {
		t.Fatalf("SaveAppliedProfileState() error = %v", err)
	}

	service := newDeploymentReviewTestService(store)
	preview, err := service.BuildDeploymentReviewPreview(context.Background(), profileID)
	if err != nil {
		t.Fatalf("BuildDeploymentReviewPreview() error = %v", err)
	}

	_, err = service.SetDeploymentDriftDecision(
		context.Background(),
		profileID,
		preview.PreviewHash,
		"Mods/SkyUI/Data/SkyUI.esp",
		"not-a-decision",
	)
	if err == nil {
		t.Fatal("SetDeploymentDriftDecision() error = nil, want invalid decision rejection")
	}
	if apperror.UserMessage(err) != "That drift decision is not allowed for this file." {
		t.Fatalf("SetDeploymentDriftDecision() error = %q, want invalid decision rejection", apperror.UserMessage(err))
	}
}
