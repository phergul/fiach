package services

import (
	"context"
	"testing"

	"github.com/phergul/fiach/internal/installconfig"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestDeploymentReviewServiceBuildsPreview(t *testing.T) {
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

	service := NewDeploymentReviewService(store, testLogger())
	preview, err := service.BuildDeploymentReviewPreview(context.Background(), profileID)
	if err != nil {
		t.Fatalf("BuildDeploymentReviewPreview() error = %v", err)
	}

	if preview.PreviewHash == "" || preview.Summary.PreviewHash != preview.PreviewHash {
		t.Fatalf("BuildDeploymentReviewPreview() hash = %q summary hash = %q, want populated stable hash", preview.PreviewHash, preview.Summary.PreviewHash)
	}
	if preview.Summary.PlanMode != "first_apply" || !preview.Summary.CanApply {
		t.Fatalf("BuildDeploymentReviewPreview() summary = %+v, want first_apply can apply", preview.Summary)
	}
	if len(preview.Root.Children) != 1 {
		t.Fatalf("BuildDeploymentReviewPreview() root children = %d, want 1", len(preview.Root.Children))
	}

	children, err := service.LoadDeploymentTreeChildren(context.Background(), preview.PreviewHash, "Mods")
	if err != nil {
		t.Fatalf("LoadDeploymentTreeChildren() error = %v", err)
	}
	if len(children) != 1 || children[0].Name != "SkyUI" {
		t.Fatalf("LoadDeploymentTreeChildren() = %+v, want SkyUI child", children)
	}

	skyUIChildren, err := service.LoadDeploymentTreeChildren(context.Background(), preview.PreviewHash, "Mods/SkyUI")
	if err != nil {
		t.Fatalf("LoadDeploymentTreeChildren(SkyUI) error = %v", err)
	}
	if len(skyUIChildren) != 1 || skyUIChildren[0].Name != "Data" || !skyUIChildren[0].IsDirectory {
		t.Fatalf("LoadDeploymentTreeChildren(SkyUI) = %+v, want Data directory child", skyUIChildren)
	}

	dataChildren, err := service.LoadDeploymentTreeChildren(context.Background(), preview.PreviewHash, "Mods/SkyUI/Data")
	if err != nil {
		t.Fatalf("LoadDeploymentTreeChildren(Data) error = %v", err)
	}
	if len(dataChildren) != 1 || dataChildren[0].Name != "SkyUI.esp" {
		t.Fatalf("LoadDeploymentTreeChildren(Data) = %+v, want SkyUI.esp child", dataChildren)
	}

	detail, err := service.GetDeploymentFileDetail(context.Background(), preview.PreviewHash, "Mods/SkyUI/Data/SkyUI.esp")
	if err != nil {
		t.Fatalf("GetDeploymentFileDetail() error = %v", err)
	}
	if detail.PlannedAction != "create" || detail.States.Desired == nil || !detail.States.Desired.Exists {
		t.Fatalf("GetDeploymentFileDetail() = %+v, want create with desired state", detail)
	}
	if detail.States.Baseline == nil || detail.States.Applied == nil {
		t.Fatalf("GetDeploymentFileDetail() states = %+v, want baseline and applied slots", detail.States)
	}
}

func TestDeploymentReviewServiceBlocksWhenProfileAlreadyApplied(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")

	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profileID,
		ManifestJSON:        `{"version":1}`,
		ProfileSnapshotJSON: `{"version":1}`,
		ProfileSnapshotHash: "snapshot",
	}); err != nil {
		t.Fatalf("SaveAppliedProfileState() error = %v", err)
	}

	service := NewDeploymentReviewService(store, testLogger())
	_, err := service.BuildDeploymentReviewPreview(context.Background(), profileID)
	if err == nil {
		t.Fatal("BuildDeploymentReviewPreview() error = nil, want applied profile gate")
	}
	if err.Error() != "Restore vanilla before applying another profile." {
		t.Fatalf("BuildDeploymentReviewPreview() error = %q, want applied profile gate", err.Error())
	}
}

func TestDeploymentReviewServiceRejectsStalePreviewHash(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	sourcePath := makeProfilePlanSourceTree(t, map[string]string{
		"plugin.txt": "content",
	})
	modID := insertServiceProfileTestMod(t, store, gameID, "Example", sourcePath)

	addServiceProfileMod(t, store, profileID, modID, true, 0)
	addServiceInstallConfig(t, store, modID, string(dto.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "plugin.txt", nil)

	service := NewDeploymentReviewService(store, testLogger())
	if _, err := service.BuildDeploymentReviewPreview(context.Background(), profileID); err != nil {
		t.Fatalf("BuildDeploymentReviewPreview() error = %v", err)
	}

	_, err := service.GetDeploymentFileDetail(context.Background(), "missing-hash", "plugin.txt")
	if err == nil {
		t.Fatal("GetDeploymentFileDetail() error = nil, want stale preview error")
	}
	if err.Error() != "The deployment preview is no longer available. Refresh the preview and try again." {
		t.Fatalf("GetDeploymentFileDetail() error = %q, want stale preview detail", err.Error())
	}
}
