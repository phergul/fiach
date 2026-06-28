package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/installconfig"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func newDeploymentReviewTestService(store *storage.Store) *DeploymentReviewService {
	profileService := NewProfileService(store, testLogger())
	return NewDeploymentReviewService(store, profileService, testLogger())
}

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

	service := newDeploymentReviewTestService(store)
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

func TestDeploymentReviewServiceSameProfileIncrementalPreviewDetectsDrift(t *testing.T) {
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

	if preview.Summary.PlanMode != "incremental" {
		t.Fatalf("BuildDeploymentReviewPreview() plan mode = %q, want incremental", preview.Summary.PlanMode)
	}
	if preview.Summary.CanApply {
		t.Fatal("BuildDeploymentReviewPreview() CanApply = true, want false for incremental preview")
	}
	if preview.Summary.StatusCounts["drifted"] != 1 {
		t.Fatalf("BuildDeploymentReviewPreview() status counts = %+v, want one drifted path", preview.Summary.StatusCounts)
	}
	if preview.Summary.AppliedAt == nil {
		t.Fatal("BuildDeploymentReviewPreview() AppliedAt = nil, want populated applied timestamp")
	}

	detail, err := service.GetDeploymentFileDetail(context.Background(), preview.PreviewHash, "Mods/SkyUI/Data/SkyUI.esp")
	if err != nil {
		t.Fatalf("GetDeploymentFileDetail() error = %v", err)
	}
	if detail.PlannedAction != "require_decision" || detail.FileStatus != "drifted" {
		t.Fatalf("GetDeploymentFileDetail() = %+v, want drifted require_decision", detail)
	}
	if detail.States.Applied == nil || !detail.States.Applied.Exists {
		t.Fatalf("GetDeploymentFileDetail() applied = %+v, want last-applied state", detail.States.Applied)
	}
	if detail.States.Current == nil || !detail.States.Current.Exists {
		t.Fatalf("GetDeploymentFileDetail() current = %+v, want current disk state", detail.States.Current)
	}
	if detail.DriftKind != "modified" {
		t.Fatalf("GetDeploymentFileDetail() DriftKind = %q, want modified", detail.DriftKind)
	}
	if detail.LastAppliedAt == nil {
		t.Fatal("GetDeploymentFileDetail() LastAppliedAt = nil, want populated")
	}
	if detail.Comparison.AppliedMatchesCurrent {
		t.Fatal("GetDeploymentFileDetail() AppliedMatchesCurrent = true, want false")
	}
	if detail.Comparison.CurrentMatchesDesired {
		t.Fatal("GetDeploymentFileDetail() CurrentMatchesDesired = true, want false")
	}
	if detail.DriftExplanation == "" {
		t.Fatal("GetDeploymentFileDetail() DriftExplanation = empty, want populated")
	}
	for _, writer := range detail.WriterStack {
		if writer.SourceKind == "base_game" {
			t.Fatalf("GetDeploymentFileDetail() writer stack = %+v, want no base_game writer for mod-added path", detail.WriterStack)
		}
	}
}

func TestDeploymentReviewServiceIncrementalModAddedPathOmitsBaseGameWriter(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	sourcePath := makeProfilePlanSourceTree(t, map[string]string{
		"recording.mov": "mod-content",
	})
	modID := insertServiceProfileTestMod(t, store, gameID, "Screenshots", sourcePath)

	addServiceProfileMod(t, store, profileID, modID, true, 0)
	addServiceInstallConfig(t, store, modID, string(dto.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Screenshots", nil)

	targetPath := filepath.Join(gameRoot, "Screenshots", "recording.mov")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("mod-content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appliedSHA256, appliedSize, err := fileops.FileIntegrity(targetPath)
	if err != nil {
		t.Fatalf("FileIntegrity() error = %v", err)
	}
	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profileID,
		ManifestJSON:        `{"version":2,"createdDirectories":[],"addedFiles":[],"replacedFiles":[],"files":{}}`,
		ProfileSnapshotJSON: `{"version":2}`,
		ProfileSnapshotHash: "snapshot",
		FileStates: []dbtypes.AppliedFileStateRow{
			{
				GameID:           gameID,
				GameRelativePath: "Screenshots/recording.mov",
				ProfileID:        profileID,
				AppliedExists:    true,
				AppliedSHA256:    &appliedSHA256,
				AppliedSizeBytes: &appliedSize,
				OutputKind:       "copied",
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

	detail, err := service.GetDeploymentFileDetail(context.Background(), preview.PreviewHash, "Screenshots/recording.mov")
	if err != nil {
		t.Fatalf("GetDeploymentFileDetail() error = %v", err)
	}
	if detail.FileStatus != "unchanged" {
		t.Fatalf("GetDeploymentFileDetail() status = %q, want unchanged", detail.FileStatus)
	}
	if detail.States.Baseline != nil && detail.States.Baseline.Exists {
		t.Fatalf("GetDeploymentFileDetail() baseline = %+v, want absent baseline", detail.States.Baseline)
	}
	if len(detail.WriterStack) != 1 {
		t.Fatalf("GetDeploymentFileDetail() writer stack = %+v, want single mod writer", detail.WriterStack)
	}
	if detail.WriterStack[0].SourceKind == "base_game" || detail.WriterStack[0].ModName != "Screenshots" {
		t.Fatalf("GetDeploymentFileDetail() writer stack = %+v, want Screenshots mod writer only", detail.WriterStack)
	}
}

func TestDeploymentReviewServiceIncrementalRemovedModPathDeletes(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")

	targetPath := filepath.Join(gameRoot, "Screenshots", "recording.mov")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("mod-content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appliedSHA256, appliedSize, err := fileops.FileIntegrity(targetPath)
	if err != nil {
		t.Fatalf("FileIntegrity() error = %v", err)
	}
	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profileID,
		ManifestJSON:        `{"version":2,"createdDirectories":[],"addedFiles":[],"replacedFiles":[],"files":{}}`,
		ProfileSnapshotJSON: `{"version":2}`,
		ProfileSnapshotHash: "snapshot",
		FileStates: []dbtypes.AppliedFileStateRow{
			{
				GameID:           gameID,
				GameRelativePath: "Screenshots/recording.mov",
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

	if preview.Summary.PlanMode != "incremental" {
		t.Fatalf("BuildDeploymentReviewPreview() plan mode = %q, want incremental", preview.Summary.PlanMode)
	}
	if !preview.Summary.CanApply {
		t.Fatal("BuildDeploymentReviewPreview() CanApply = false, want true for actionable delete preview")
	}
	if preview.Summary.StatusCounts["deleted"] != 1 {
		t.Fatalf("BuildDeploymentReviewPreview() status counts = %+v, want one deleted path", preview.Summary.StatusCounts)
	}

	detail, err := service.GetDeploymentFileDetail(context.Background(), preview.PreviewHash, "Screenshots/recording.mov")
	if err != nil {
		t.Fatalf("GetDeploymentFileDetail() error = %v", err)
	}
	if detail.PlannedAction != "delete" || detail.FileStatus != "deleted" {
		t.Fatalf("GetDeploymentFileDetail() = %+v, want deleted delete", detail)
	}
	if detail.States.Desired != nil && detail.States.Desired.Exists {
		t.Fatalf("GetDeploymentFileDetail() desired = %+v, want absent desired state", detail.States.Desired)
	}
}

func TestDeploymentReviewServiceBlocksDifferentAppliedProfile(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	appliedProfileID := insertServiceProfileTestProfile(t, store, gameID, "Applied")
	otherProfileID := insertServiceProfileTestProfile(t, store, gameID, "Other")

	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           appliedProfileID,
		ManifestJSON:        `{"version":1}`,
		ProfileSnapshotJSON: `{"version":1}`,
		ProfileSnapshotHash: "snapshot",
	}); err != nil {
		t.Fatalf("SaveAppliedProfileState() error = %v", err)
	}

	service := newDeploymentReviewTestService(store)
	_, err := service.BuildDeploymentReviewPreview(context.Background(), otherProfileID)
	if err == nil {
		t.Fatal("BuildDeploymentReviewPreview() error = nil, want different applied profile gate")
	}
	if apperror.UserMessage(err) != "Restore vanilla before applying another profile." {
		t.Fatalf("BuildDeploymentReviewPreview() error = %q, want applied profile gate", apperror.UserMessage(err))
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

	service := newDeploymentReviewTestService(store)
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

func TestDeploymentReviewServiceApplyIncrementalDeploymentDeletesRemovedPath(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")

	targetPath := filepath.Join(gameRoot, "Screenshots", "recording.mov")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("mod-content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appliedSHA256, appliedSize, err := fileops.FileIntegrity(targetPath)
	if err != nil {
		t.Fatalf("FileIntegrity() error = %v", err)
	}
	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profileID,
		ManifestJSON:        `{"version":2,"createdDirectories":[],"addedFiles":[],"replacedFiles":[],"files":{}}`,
		ProfileSnapshotJSON: `{"version":2}`,
		ProfileSnapshotHash: "snapshot",
		FileStates: []dbtypes.AppliedFileStateRow{
			{
				GameID:           gameID,
				GameRelativePath: "Screenshots/recording.mov",
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

	result, err := service.ApplyIncrementalDeployment(context.Background(), profileID, preview.PreviewHash)
	if err != nil {
		t.Fatalf("ApplyIncrementalDeployment() error = %v", err)
	}
	if !result.Success || result.CompletedCount != 1 {
		t.Fatalf("ApplyIncrementalDeployment() = %+v, want successful delete", result)
	}

	if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) {
		t.Fatalf("target after apply stat = %v, want deleted file", statErr)
	}

	hasRows, err := store.HasAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("HasAppliedFileStates() error = %v", err)
	}
	if hasRows {
		rows, err := store.ListAppliedFileStates(context.Background(), gameID)
		if err != nil {
			t.Fatalf("ListAppliedFileStates() error = %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("applied file states = %+v, want empty after delete", rows)
		}
	}
}

func TestDeploymentReviewServiceApplyIncrementalDeploymentRejectsStalePreviewHash(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")

	targetPath := filepath.Join(gameRoot, "Screenshots", "recording.mov")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("mod-content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appliedSHA256, appliedSize, err := fileops.FileIntegrity(targetPath)
	if err != nil {
		t.Fatalf("FileIntegrity() error = %v", err)
	}
	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profileID,
		ManifestJSON:        `{"version":2,"createdDirectories":[],"addedFiles":[],"replacedFiles":[],"files":{}}`,
		ProfileSnapshotJSON: `{"version":2}`,
		ProfileSnapshotHash: "snapshot",
		FileStates: []dbtypes.AppliedFileStateRow{
			{
				GameID:           gameID,
				GameRelativePath: "Screenshots/recording.mov",
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
	if _, err := service.BuildDeploymentReviewPreview(context.Background(), profileID); err != nil {
		t.Fatalf("BuildDeploymentReviewPreview() error = %v", err)
	}

	_, err = service.ApplyIncrementalDeployment(context.Background(), profileID, "stale-hash")
	if err == nil {
		t.Fatal("ApplyIncrementalDeployment() error = nil, want stale preview hash")
	}
	if err.Error() != "The deployment preview is stale. Refresh the preview and try again." {
		t.Fatalf("ApplyIncrementalDeployment() error = %q, want stale preview hash", err.Error())
	}
}
