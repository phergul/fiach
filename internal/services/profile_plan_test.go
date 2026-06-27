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

	"github.com/phergul/fiach/internal/apperror"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/installconfig"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage"
	"github.com/phergul/fiach/internal/storage/dbtypes"
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
	addServiceInstallConfig(t, store, modID, string(dto.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Mods/SkyUI", nil)

	service := NewProfileService(store, testLogger())
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
	addServiceInstallConfig(t, store, modID, string(dto.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Data", nil)

	service := NewProfileService(store, testLogger())
	plan, err := service.BuildProfileOperationPlan(context.Background(), profileID)
	if err != nil {
		t.Fatalf("BuildProfileOperationPlan() error = %v, want planner issue result", err)
	}

	if plan.CanApply {
		t.Fatalf("BuildProfileOperationPlan() CanApply = true, want false: %+v", plan)
	}
	if !servicePlanHasIssueKind(plan.Issues, dto.PlanIssueMissingSourceRoot) {
		t.Fatalf("BuildProfileOperationPlan() issues = %+v, want missing source root issue", plan.Issues)
	}
}

func TestProfileServiceWrapsUnexpectedPlannerErrors(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewProfileService(store, testLogger())
	_, err := service.BuildProfileOperationPlan(context.Background(), 999)
	if err == nil {
		t.Fatal("BuildProfileOperationPlan() error = nil, want planner error")
	}
	if err.Error() != "Profile was not found." {
		t.Fatalf("BuildProfileOperationPlan() error = %q, want missing profile detail", err.Error())
	}
	if !strings.Contains(apperror.Detail(err), "resolve profile plan") {
		t.Fatalf("BuildProfileOperationPlan() detail = %q, want resolver context", apperror.Detail(err))
	}
}

func TestProfileServiceApplyProfileOperationPlanExecutesPreviewedPlan(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	modID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", "/managed/skyui")
	addServiceProfileMod(t, store, profileID, modID, true, 0)
	sourcePath := makeProfilePlanSourceTree(t, map[string]string{
		"Data/modded.txt": "modded",
	})
	targetPath := filepath.Join(gameRoot, "Data", "modded.txt")
	backupPath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(dbtypes.StoredGame{ID: gameID}), "operation-backups", "Data", "modded.txt")
	sourceFilePath := filepath.Join(sourcePath, "Data", "modded.txt")

	service := NewProfileService(store, testLogger())
	result, err := service.ApplyProfileOperationPlan(context.Background(), profileID, dto.OperationPlan{
		CanApply: true,
		Operations: []dto.Operation{
			{
				Type:       dto.OperationTypeCreateDirectory,
				TargetPath: filepath.Dir(targetPath),
			},
			{
				Type:       dto.OperationTypeCopy,
				SourcePath: &sourceFilePath,
				TargetPath: targetPath,
				BackupPath: &backupPath,
				Mod: dto.ModContext{
					ModID:   modID,
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
	if manifest.Version != appliedstate.DocumentVersionV2 || len(manifest.CreatedDirectories) != 1 || len(manifest.AddedFiles) != 1 {
		t.Fatalf("manifest JSON = %+v, want v2 created directory and added file", manifest)
	}
	if len(manifest.Files) != 1 {
		t.Fatalf("manifest files map = %+v, want one v2 file entry", manifest.Files)
	}

	fileStates, err := store.ListAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListAppliedFileStates() error = %v", err)
	}
	if len(fileStates) != 1 || fileStates[0].GameRelativePath != "Data/modded.txt" || fileStates[0].AppliedSHA256 == nil || *fileStates[0].AppliedSHA256 == "" {
		t.Fatalf("ListAppliedFileStates() = %+v, want persisted added file state", fileStates)
	}
	if manifest.AddedFiles[0].TargetPath != targetPath || manifest.AddedFiles[0].SHA256 == "" || manifest.AddedFiles[0].SizeBytes != int64(len("modded")) {
		t.Fatalf("manifest added file = %+v, want target integrity", manifest.AddedFiles[0])
	}
	if manifest.AddedFiles[0].Mod.ID != modID || manifest.AddedFiles[0].Mod.Name != "SkyUI" {
		t.Fatalf("manifest added file mod = %+v, want tagged mod document", manifest.AddedFiles[0].Mod)
	}

	var snapshot appliedstate.ProfileSnapshotDocument
	if err := json.Unmarshal([]byte(state.ProfileSnapshotJSON), &snapshot); err != nil {
		t.Fatalf("unmarshal profile snapshot JSON: %v", err)
	}
	if snapshot.Version != appliedstate.DocumentVersion || !snapshot.CanApply || len(snapshot.Operations) != 2 {
		t.Fatalf("profile snapshot JSON = %+v, want two applicable operations", snapshot)
	}
	if snapshot.Operations[1].Mod.ID != modID || snapshot.Operations[1].Mod.Name != "SkyUI" {
		t.Fatalf("profile snapshot operation mod = %+v, want tagged mod document", snapshot.Operations[1].Mod)
	}
	if state.ProfileSnapshotHash != sha256Hex(state.ProfileSnapshotJSON) {
		t.Fatalf("profile snapshot hash = %q, want SHA-256 of snapshot JSON", state.ProfileSnapshotHash)
	}

	var compositionSnapshot appliedstate.ProfileCompositionDocument
	if state.ProfileCompositionSnapshotJSON == nil {
		t.Fatal("profile composition snapshot JSON = nil, want persisted composition snapshot")
	}
	if err := json.Unmarshal([]byte(*state.ProfileCompositionSnapshotJSON), &compositionSnapshot); err != nil {
		t.Fatalf("unmarshal profile composition snapshot JSON: %v", err)
	}
	if compositionSnapshot.Version != appliedstate.DocumentVersion || compositionSnapshot.ProfileID != profileID || len(compositionSnapshot.Mods) != 1 {
		t.Fatalf("profile composition snapshot JSON = %+v, want applied profile composition", compositionSnapshot)
	}
	if compositionSnapshot.Mods[0].ModID != modID || !compositionSnapshot.Mods[0].Enabled || compositionSnapshot.Mods[0].LoadOrder != 0 {
		t.Fatalf("profile composition snapshot mod = %+v, want profile mod state", compositionSnapshot.Mods[0])
	}
	if state.ProfileCompositionSnapshotHash == nil || *state.ProfileCompositionSnapshotHash != sha256Hex(*state.ProfileCompositionSnapshotJSON) {
		t.Fatalf("profile composition snapshot hash = %v, want SHA-256 of snapshot JSON", state.ProfileCompositionSnapshotHash)
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

	service := NewProfileService(store, testLogger())
	result, err := service.ApplyProfileOperationPlan(context.Background(), profileID, dto.OperationPlan{
		CanApply: true,
		Operations: []dto.Operation{
			{
				Type:       dto.OperationTypeCopy,
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

func TestProfileServiceApplyProfileOperationPlanRejectsWhenProfileAlreadyApplied(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	firstProfileID := insertServiceProfileTestProfile(t, store, gameID, "First")
	secondProfileID := insertServiceProfileTestProfile(t, store, gameID, "Second")
	service := NewProfileService(store, testLogger())

	firstSourceRoot := makeProfilePlanSourceTree(t, map[string]string{"first.txt": "first"})
	firstSourcePath := filepath.Join(firstSourceRoot, "first.txt")
	firstTargetPath := filepath.Join(gameRoot, "first.txt")
	firstResult, err := service.ApplyProfileOperationPlan(context.Background(), firstProfileID, dto.OperationPlan{
		CanApply: true,
		Operations: []dto.Operation{
			{
				Type:       dto.OperationTypeCopy,
				SourcePath: &firstSourcePath,
				TargetPath: firstTargetPath,
				Mod: dto.ModContext{
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

	secondSourceRoot := makeProfilePlanSourceTree(t, map[string]string{"second.txt": "second"})
	secondSourcePath := filepath.Join(secondSourceRoot, "second.txt")
	secondTargetPath := filepath.Join(gameRoot, "second.txt")
	blockedResult, err := service.ApplyProfileOperationPlan(context.Background(), secondProfileID, dto.OperationPlan{
		CanApply: true,
		Operations: []dto.Operation{
			{
				Type:       dto.OperationTypeCopy,
				SourcePath: &secondSourcePath,
				TargetPath: secondTargetPath,
			},
		},
	})
	if err == nil {
		t.Fatal("ApplyProfileOperationPlan() second error = nil, want applied-state guard")
	}
	if blockedResult.Success || blockedResult.CompletedCount != 0 {
		t.Fatalf("ApplyProfileOperationPlan() blocked result = %+v, want empty failure result", blockedResult)
	}
	if err.Error() != "Restore vanilla before applying another profile." {
		t.Fatalf("ApplyProfileOperationPlan() second error = %q, want applied-state guard", err.Error())
	}
	if _, err := os.Stat(secondTargetPath); !os.IsNotExist(err) {
		t.Fatalf("second target stat error = %v, want no file written", err)
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
	if _, err := store.DB().Exec(`
		CREATE TRIGGER fail_applied_profile_state_insert
		BEFORE INSERT ON applied_profile_states
		BEGIN
			SELECT RAISE(ABORT, 'forced applied state failure');
		END
	`); err != nil {
		t.Fatalf("create failing trigger: %v", err)
	}

	sourceRoot := makeProfilePlanSourceTree(t, map[string]string{"modded.txt": "modded"})
	sourcePath := filepath.Join(sourceRoot, "modded.txt")
	targetPath := filepath.Join(gameRoot, "modded.txt")
	service := NewProfileService(store, testLogger())
	result, err := service.ApplyProfileOperationPlan(context.Background(), profileID, dto.OperationPlan{
		CanApply: true,
		Operations: []dto.Operation{
			{
				Type:       dto.OperationTypeCopy,
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
	if !strings.Contains(apperror.Detail(err), "save applied profile state") || !strings.Contains(apperror.Detail(err), "forced applied state failure") {
		t.Fatalf("ApplyProfileOperationPlan() error = %q, want persistence detail", apperror.Detail(err))
	}
	assertServiceFileContents(t, targetPath, "modded")

	_, found, readErr := store.GetAppliedProfileState(context.Background(), gameID)
	if readErr != nil {
		t.Fatalf("GetAppliedProfileState() error = %v", readErr)
	}
	if found {
		t.Fatal("GetAppliedProfileState() found = true, want no state after persistence failure")
	}
}

func TestProfileServiceApplyProfileOperationPlanRejectsBlockingIssues(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewProfileService(store, testLogger())
	_, err := service.ApplyProfileOperationPlan(context.Background(), 1, dto.OperationPlan{CanApply: false})
	if err == nil {
		t.Fatal("ApplyProfileOperationPlan() error = nil, want blocking issue error")
	}
	if err.Error() != "Fix the issues in the plan before applying." {
		t.Fatalf("ApplyProfileOperationPlan() error = %q, want blocking issue detail", err.Error())
	}
}

func TestProfileServiceApplyProfileOperationPlanRejectsInvalidProfileID(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewProfileService(store, testLogger())
	_, err := service.ApplyProfileOperationPlan(context.Background(), 0, dto.OperationPlan{CanApply: true})
	if err == nil {
		t.Fatal("ApplyProfileOperationPlan() error = nil, want invalid profile ID error")
	}
	if err.Error() != "A valid profile must be selected." {
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

func servicePlanHasIssueKind(issues []dto.PlanIssue, kind dto.PlanIssueKind) bool {
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

func TestProfileServiceLoadAppliedFileStatesLazilyMigratesV1Manifest(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	targetPath := filepath.Join(gameRoot, "Data", "modded.txt")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("create target directory: %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("modded"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}

	manifestJSON, err := appliedstate.EncodeManifest(appliedstate.ManifestDocument{
		Version: appliedstate.DocumentVersionV1,
		AddedFiles: []appliedstate.AddedFile{
			{
				OperationIndex: 0,
				Mod:            appliedstate.Mod{ID: 10, Name: "SkyUI"},
				TargetPath:     targetPath,
				SHA256:         "added-sha",
				SizeBytes:      6,
			},
		},
	})
	if err != nil {
		t.Fatalf("EncodeManifest() error = %v", err)
	}
	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profileID,
		ManifestJSON:        manifestJSON,
		ProfileSnapshotJSON: `{"version":1}`,
		ProfileSnapshotHash: "hash",
	}); err != nil {
		t.Fatalf("SaveAppliedProfileState() error = %v", err)
	}

	service := NewProfileService(store, testLogger())
	states, err := service.LoadAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("LoadAppliedFileStates() error = %v", err)
	}
	if len(states) != 1 || states[0].GameRelativePath != "Data/modded.txt" || states[0].AppliedSHA256 == nil || *states[0].AppliedSHA256 != "added-sha" {
		t.Fatalf("LoadAppliedFileStates() = %+v, want migrated added file state", states)
	}

	found, err := store.HasAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("HasAppliedFileStates() error = %v", err)
	}
	if !found {
		t.Fatal("HasAppliedFileStates() found = false, want lazy migration persisted rows")
	}

	statesAgain, err := service.LoadAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("LoadAppliedFileStates() second error = %v", err)
	}
	if len(statesAgain) != 1 {
		t.Fatalf("LoadAppliedFileStates() second = %+v, want idempotent load", statesAgain)
	}
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
