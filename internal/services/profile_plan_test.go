package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/mod-manager/internal/appliedstate"
	"github.com/phergul/mod-manager/internal/installconfig"
	"github.com/phergul/mod-manager/internal/operationplan"
	"github.com/phergul/mod-manager/internal/storage"
)

func TestProfileServiceBuildsProfileOperationPlan(t *testing.T) {
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
	addServiceInstallConfig(t, store, modID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Mods/SkyUI", nil)

	service := NewProfileService(store)
	plan, err := service.BuildProfileOperationPlan(context.Background(), profileID)
	if err != nil {
		t.Fatalf("BuildProfileOperationPlan() error = %v", err)
	}

	if !plan.CanApply || len(plan.Issues) != 0 || len(plan.Operations) == 0 {
		t.Fatalf("BuildProfileOperationPlan() = %+v, want applicable plan with operations", plan)
	}
}

func TestProfileServiceReturnsPlannerIssuesInPlanResult(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	modID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", filepath.Join(t.TempDir(), "missing"))

	addServiceProfileMod(t, store, profileID, modID, true, 0)
	addServiceInstallConfig(t, store, modID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Data", nil)

	service := NewProfileService(store)
	plan, err := service.BuildProfileOperationPlan(context.Background(), profileID)
	if err != nil {
		t.Fatalf("BuildProfileOperationPlan() error = %v, want planner issue result", err)
	}

	if plan.CanApply {
		t.Fatalf("BuildProfileOperationPlan() CanApply = true, want false: %+v", plan)
	}
	if !servicePlanHasIssueKind(plan.Issues, operationplan.PlanIssueMissingSourceRoot) {
		t.Fatalf("BuildProfileOperationPlan() issues = %+v, want missing source root issue", plan.Issues)
	}
}

func TestProfileServiceWrapsUnexpectedPlannerErrors(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewProfileService(store)
	_, err := service.BuildProfileOperationPlan(context.Background(), 999)
	if err == nil {
		t.Fatal("BuildProfileOperationPlan() error = nil, want planner error")
	}
	if !strings.Contains(err.Error(), "build profile operation plan") || !strings.Contains(err.Error(), "resolve profile plan") || !strings.Contains(err.Error(), "profile 999 was not found") {
		t.Fatalf("BuildProfileOperationPlan() error = %q, want wrapped resolver detail", err.Error())
	}
}

func TestProfileServiceApplyProfileOperationPlanExecutesPreviewedPlan(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	sourcePath := makeProfilePlanSourceTree(t, map[string]string{
		"Data/modded.txt": "modded",
	})
	targetPath := filepath.Join(gameRoot, "Data", "modded.txt")
	backupPath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(storage.StoredGame{ID: gameID}), "operation-backups", "Data", "modded.txt")
	sourceFilePath := filepath.Join(sourcePath, "Data", "modded.txt")

	service := NewProfileService(store)
	result, err := service.ApplyProfileOperationPlan(context.Background(), profileID, operationplan.OperationPlan{
		CanApply: true,
		Operations: []operationplan.Operation{
			{
				Type:       operationplan.OperationTypeCreateDirectory,
				TargetPath: filepath.Dir(targetPath),
			},
			{
				Type:       operationplan.OperationTypeCopy,
				SourcePath: &sourceFilePath,
				TargetPath: targetPath,
				BackupPath: &backupPath,
				Mod: operationplan.ModContext{
					ModID:   1,
					ModName: "SkyUI",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyProfileOperationPlan() error = %v", err)
	}
	if !result.Success || result.CompletedCount != 2 {
		t.Fatalf("ApplyProfileOperationPlan() = %+v, want two completed operations", result)
	}
	contents, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", targetPath, err)
	}
	if string(contents) != "modded" {
		t.Fatalf("ApplyProfileOperationPlan() wrote %q, want modded", contents)
	}

	state, found, err := store.GetAppliedProfileState(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetAppliedProfileState() error = %v", err)
	}
	if !found {
		t.Fatal("GetAppliedProfileState() found = false, want persisted apply state")
	}
	if state.GameID != gameID || state.ProfileID != profileID || state.AppliedAt == "" {
		t.Fatalf("applied profile state = %+v, want game/profile linkage and applied timestamp", state)
	}

	var manifest appliedstate.ManifestDocument
	if err := json.Unmarshal([]byte(state.ManifestJSON), &manifest); err != nil {
		t.Fatalf("unmarshal manifest JSON: %v", err)
	}
	if manifest.Version != appliedstate.DocumentVersion || len(manifest.CreatedDirectories) != 1 || len(manifest.AddedFiles) != 1 {
		t.Fatalf("manifest JSON = %+v, want created directory and added file", manifest)
	}
	if manifest.AddedFiles[0].TargetPath != targetPath || manifest.AddedFiles[0].SHA256 == "" || manifest.AddedFiles[0].SizeBytes != int64(len("modded")) {
		t.Fatalf("manifest added file = %+v, want target integrity", manifest.AddedFiles[0])
	}
	if manifest.AddedFiles[0].Mod.ID != 1 || manifest.AddedFiles[0].Mod.Name != "SkyUI" {
		t.Fatalf("manifest added file mod = %+v, want tagged mod document", manifest.AddedFiles[0].Mod)
	}

	var snapshot appliedstate.ProfileSnapshotDocument
	if err := json.Unmarshal([]byte(state.ProfileSnapshotJSON), &snapshot); err != nil {
		t.Fatalf("unmarshal profile snapshot JSON: %v", err)
	}
	if snapshot.Version != appliedstate.DocumentVersion || !snapshot.CanApply || len(snapshot.Operations) != 2 {
		t.Fatalf("profile snapshot JSON = %+v, want two applicable operations", snapshot)
	}
	if snapshot.Operations[1].Mod.ID != 1 || snapshot.Operations[1].Mod.Name != "SkyUI" {
		t.Fatalf("profile snapshot operation mod = %+v, want tagged mod document", snapshot.Operations[1].Mod)
	}
	if state.ProfileSnapshotHash != sha256Hex(state.ProfileSnapshotJSON) {
		t.Fatalf("profile snapshot hash = %q, want SHA-256 of snapshot JSON", state.ProfileSnapshotHash)
	}
}

func TestProfileServiceApplyProfileOperationPlanReturnsPartialResult(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	missingSourcePath := filepath.Join(t.TempDir(), "missing.txt")
	targetPath := filepath.Join(gameRoot, "Data", "missing.txt")

	service := NewProfileService(store)
	result, err := service.ApplyProfileOperationPlan(context.Background(), profileID, operationplan.OperationPlan{
		CanApply: true,
		Operations: []operationplan.Operation{
			{
				Type:       operationplan.OperationTypeCopy,
				SourcePath: &missingSourcePath,
				TargetPath: targetPath,
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyProfileOperationPlan() error = %v, want partial result", err)
	}
	if result.Success || result.FailedCount != 1 || result.Results[0].Error == nil {
		t.Fatalf("ApplyProfileOperationPlan() = %+v, want failed result", result)
	}
	_, found, err := store.GetAppliedProfileState(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetAppliedProfileState() error = %v", err)
	}
	if found {
		t.Fatal("GetAppliedProfileState() found = true, want failed apply to leave no state")
	}
}

func TestProfileServiceApplyProfileOperationPlanReplacesExistingStateOnlyAfterSuccess(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	firstProfileID := insertServiceProfileTestProfile(t, store, gameID, "First")
	secondProfileID := insertServiceProfileTestProfile(t, store, gameID, "Second")
	service := NewProfileService(store)

	firstSourceRoot := makeProfilePlanSourceTree(t, map[string]string{"first.txt": "first"})
	firstSourcePath := filepath.Join(firstSourceRoot, "first.txt")
	firstTargetPath := filepath.Join(gameRoot, "first.txt")
	firstResult, err := service.ApplyProfileOperationPlan(context.Background(), firstProfileID, operationplan.OperationPlan{
		CanApply: true,
		Operations: []operationplan.Operation{
			{
				Type:       operationplan.OperationTypeCopy,
				SourcePath: &firstSourcePath,
				TargetPath: firstTargetPath,
				Mod: operationplan.ModContext{
					ModID:   1,
					ModName: "First Mod",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyProfileOperationPlan() first error = %v", err)
	}
	if !firstResult.Success {
		t.Fatalf("ApplyProfileOperationPlan() first = %+v, want success", firstResult)
	}

	missingSourcePath := filepath.Join(t.TempDir(), "missing.txt")
	failedResult, err := service.ApplyProfileOperationPlan(context.Background(), secondProfileID, operationplan.OperationPlan{
		CanApply: true,
		Operations: []operationplan.Operation{
			{
				Type:       operationplan.OperationTypeCopy,
				SourcePath: &missingSourcePath,
				TargetPath: filepath.Join(gameRoot, "missing.txt"),
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyProfileOperationPlan() failed apply error = %v, want partial result only", err)
	}
	if failedResult.Success {
		t.Fatalf("ApplyProfileOperationPlan() failed apply = %+v, want failure", failedResult)
	}

	state, found, err := store.GetAppliedProfileState(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetAppliedProfileState() error = %v", err)
	}
	if !found || state.ProfileID != firstProfileID {
		t.Fatalf("applied profile state = %+v found=%v, want original first profile state", state, found)
	}
}

func TestProfileServiceApplyProfileOperationPlanReturnsResultWhenStatePersistenceFails(t *testing.T) {
	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	if _, err := store.SaveAppliedProfileState(context.Background(), storage.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profileID,
		ManifestJSON:        `{"version":1}`,
		ProfileSnapshotJSON: `{"version":1}`,
		ProfileSnapshotHash: "previous",
	}); err != nil {
		t.Fatalf("SaveAppliedProfileState() setup error = %v", err)
	}
	if _, err := store.DB().Exec(`
		CREATE TRIGGER fail_applied_profile_state_update
		BEFORE UPDATE ON applied_profile_states
		BEGIN
			SELECT RAISE(ABORT, 'forced applied state failure');
		END
	`); err != nil {
		t.Fatalf("create failing trigger: %v", err)
	}

	sourceRoot := makeProfilePlanSourceTree(t, map[string]string{"modded.txt": "modded"})
	sourcePath := filepath.Join(sourceRoot, "modded.txt")
	targetPath := filepath.Join(gameRoot, "modded.txt")
	service := NewProfileService(store)
	result, err := service.ApplyProfileOperationPlan(context.Background(), profileID, operationplan.OperationPlan{
		CanApply: true,
		Operations: []operationplan.Operation{
			{
				Type:       operationplan.OperationTypeCopy,
				SourcePath: &sourcePath,
				TargetPath: targetPath,
			},
		},
	})
	if err == nil {
		t.Fatal("ApplyProfileOperationPlan() error = nil, want persistence error")
	}
	if !result.Success || result.CompletedCount != 1 {
		t.Fatalf("ApplyProfileOperationPlan() result = %+v, want populated successful apply result", result)
	}
	if !strings.Contains(err.Error(), "apply profile operation plan") || !strings.Contains(err.Error(), "save applied profile state") || !strings.Contains(err.Error(), "forced applied state failure") {
		t.Fatalf("ApplyProfileOperationPlan() error = %q, want service and persistence detail", err.Error())
	}
	assertServiceFileContents(t, targetPath, "modded")

	state, found, readErr := store.GetAppliedProfileState(context.Background(), gameID)
	if readErr != nil {
		t.Fatalf("GetAppliedProfileState() error = %v", readErr)
	}
	if !found || state.ProfileSnapshotHash != "previous" {
		t.Fatalf("applied profile state = %+v found=%v, want previous state unchanged", state, found)
	}
}

func TestProfileServiceApplyProfileOperationPlanRejectsBlockingIssues(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewProfileService(store)
	_, err := service.ApplyProfileOperationPlan(context.Background(), 1, operationplan.OperationPlan{CanApply: false})
	if err == nil {
		t.Fatal("ApplyProfileOperationPlan() error = nil, want blocking issue error")
	}
	if !strings.Contains(err.Error(), "apply profile operation plan") || !strings.Contains(err.Error(), "operation plan has blocking issues") {
		t.Fatalf("ApplyProfileOperationPlan() error = %q, want blocking issue detail", err.Error())
	}
}

func TestProfileServiceApplyProfileOperationPlanRejectsInvalidProfileID(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewProfileService(store)
	_, err := service.ApplyProfileOperationPlan(context.Background(), 0, operationplan.OperationPlan{CanApply: true})
	if err == nil {
		t.Fatal("ApplyProfileOperationPlan() error = nil, want invalid profile ID error")
	}
	if !strings.Contains(err.Error(), "apply profile operation plan") || !strings.Contains(err.Error(), "profile ID must be positive") {
		t.Fatalf("ApplyProfileOperationPlan() error = %q, want profile ID detail", err.Error())
	}
}

func insertServiceProfileTestProfile(t *testing.T, store *storage.Store, gameID int64, name string) int64 {
	t.Helper()

	result, err := store.DB().Exec(`
		INSERT INTO profiles (game_id, name)
		VALUES (?, ?)
	`, gameID, name)
	if err != nil {
		t.Fatalf("insert service profile plan test profile: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("service profile plan test profile LastInsertId(): %v", err)
	}

	return id
}

func addServiceProfileMod(t *testing.T, store *storage.Store, profileID int64, modID int64, enabled bool, loadOrder int64) {
	t.Helper()

	enabledValue := 0
	if enabled {
		enabledValue = 1
	}

	if _, err := store.DB().Exec(`
		INSERT INTO profile_mods (profile_id, mod_id, enabled, load_order)
		VALUES (?, ?, ?, ?)
	`, profileID, modID, enabledValue, loadOrder); err != nil {
		t.Fatalf("insert service profile plan test profile mod: %v", err)
	}
}

func addServiceInstallConfig(t *testing.T, store *storage.Store, modID int64, strategyType string, targetBase string, targetRelativePath string, sourceSubpath *string) {
	t.Helper()

	if _, err := store.DB().Exec(`
		INSERT INTO mod_install_configs (mod_id, strategy_type, target_base, target_relative_path, source_subpath)
		VALUES (?, ?, ?, ?, ?)
	`, modID, strategyType, targetBase, targetRelativePath, sourceSubpath); err != nil {
		t.Fatalf("insert service profile plan test config: %v", err)
	}
}

func servicePlanHasIssueKind(issues []operationplan.PlanIssue, kind operationplan.PlanIssueKind) bool {
	for _, issue := range issues {
		if issue.Kind == kind {
			return true
		}
	}

	return false
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func assertServiceFileContents(t *testing.T, path string, want string) {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	if string(contents) != want {
		t.Fatalf("os.ReadFile(%q) = %q, want %q", path, contents, want)
	}
}

func makeProfilePlanSourceTree(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for relativePath, contents := range files {
		path := filepath.Join(root, relativePath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", path, err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", path, err)
		}
	}

	return root
}
