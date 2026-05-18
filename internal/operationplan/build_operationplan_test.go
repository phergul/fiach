package operationplan

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/mod-manager/internal/installconfig"
)

func TestBuildOperationPlanCreatesDirectoriesAndFilesInStableOrder(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertPlannerGame(t, store, "Skyrim", gameRoot)
	profileID := insertPlannerProfile(t, store, gameID, "Default")

	firstSource := makePlannerSourceTree(t, map[string]string{
		"Data/First.esp": "first",
	})
	secondSource := makePlannerSourceTree(t, map[string]string{
		"plugins/core/Second.dll": "second",
	})

	firstModID := insertPlannerMod(t, store, gameID, "First", firstSource)
	secondModID := insertPlannerMod(t, store, gameID, "Second", secondSource)

	addPlannerProfileMod(t, store, profileID, firstModID, true, 0)
	addPlannerProfileMod(t, store, profileID, secondModID, true, 1)

	addPlannerInstallConfig(t, store, firstModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Mods/First", nil)
	sourceSubpath := "plugins/core"
	addPlannerInstallConfig(t, store, secondModID, string(installconfig.StrategyTypeBepInEx), installconfig.TargetBaseGameRoot, "BepInEx/plugins", &sourceSubpath)

	beforeProfileMods := countPlannerRows(t, store, "profile_mods")
	beforeConfigs := countPlannerRows(t, store, "mod_install_configs")

	resolved, err := ResolveProfilePlan(context.Background(), store, profileID)
	if err != nil {
		t.Fatalf("ResolveProfilePlan() error = %v", err)
	}

	plan, err := BuildOperationPlan(resolved)
	if err != nil {
		t.Fatalf("BuildOperationPlan() error = %v", err)
	}

	if len(plan.Operations) != 7 {
		t.Fatalf("BuildOperationPlan() operation count = %d, want 7: %+v", len(plan.Operations), plan.Operations)
	}

	wantDirTargets := []string{
		filepath.Join(gameRoot, "BepInEx"),
		filepath.Join(gameRoot, "Mods"),
		filepath.Join(gameRoot, "BepInEx", "plugins"),
		filepath.Join(gameRoot, "Mods", "First"),
		filepath.Join(gameRoot, "Mods", "First", "Data"),
	}
	for index, want := range wantDirTargets {
		operation := plan.Operations[index]
		if operation.Type != OperationTypeCreateDirectory || operation.TargetPath != want {
			t.Fatalf("directory operation %d = %+v, want create_directory %q", index, operation, want)
		}
	}

	firstFile := plan.Operations[5]
	secondFile := plan.Operations[6]
	if firstFile.Type != OperationTypeCopy || firstFile.Mod.ModID != firstModID || firstFile.SourcePath == nil || !strings.HasSuffix(*firstFile.SourcePath, filepath.Join("Data", "First.esp")) || firstFile.TargetPath != filepath.Join(gameRoot, "Mods", "First", "Data", "First.esp") {
		t.Fatalf("first file operation = %+v, want stable first mod copy operation", firstFile)
	}
	if secondFile.Type != OperationTypeCopy || secondFile.Mod.ModID != secondModID || secondFile.SourcePath == nil || !strings.HasSuffix(*secondFile.SourcePath, "Second.dll") || secondFile.TargetPath != filepath.Join(gameRoot, "BepInEx", "plugins", "Second.dll") {
		t.Fatalf("second file operation = %+v, want second mod copy operation using source subpath", secondFile)
	}

	afterProfileMods := countPlannerRows(t, store, "profile_mods")
	afterConfigs := countPlannerRows(t, store, "mod_install_configs")
	if beforeProfileMods != afterProfileMods || beforeConfigs != afterConfigs {
		t.Fatalf("row counts changed after planner: profile_mods %d->%d, mod_install_configs %d->%d", beforeProfileMods, afterProfileMods, beforeConfigs, afterConfigs)
	}
}

func TestBuildOperationPlanMarksExistingTargetsAsReplaceWithManagedBackupPath(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertPlannerGame(t, store, "Skyrim", gameRoot)
	profileID := insertPlannerProfile(t, store, gameID, "Default")

	sourcePath := makePlannerSourceTree(t, map[string]string{
		"Data/SkyUI.esp": "replacement",
	})
	modID := insertPlannerMod(t, store, gameID, "SkyUI", sourcePath)
	addPlannerProfileMod(t, store, profileID, modID, true, 0)
	addPlannerInstallConfig(t, store, modID, string(installconfig.StrategyTypeReplaceFiles), installconfig.TargetBaseGameRoot, ".", nil)

	targetPath := filepath.Join(gameRoot, "Data", "SkyUI.esp")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("vanilla"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	resolved, err := ResolveProfilePlan(context.Background(), store, profileID)
	if err != nil {
		t.Fatalf("ResolveProfilePlan() error = %v", err)
	}

	plan, err := BuildOperationPlan(resolved)
	if err != nil {
		t.Fatalf("BuildOperationPlan() error = %v", err)
	}

	if len(plan.Operations) != 1 {
		t.Fatalf("BuildOperationPlan() operation count = %d, want 1", len(plan.Operations))
	}

	operation := plan.Operations[0]
	wantBackupPath := filepath.Join(resolved.GameModStoragePath, backupRootDirName, "Data", "SkyUI.esp")
	if operation.Type != OperationTypeReplace || !operation.Conflict || operation.BackupPath == nil || *operation.BackupPath != wantBackupPath {
		t.Fatalf("replace operation = %+v, want replace with managed backup path %q", operation, wantBackupPath)
	}
}

func TestBuildOperationPlanDeduplicatesSharedDirectoriesToFirstMod(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertPlannerGame(t, store, "Skyrim", gameRoot)
	profileID := insertPlannerProfile(t, store, gameID, "Default")

	firstSource := makePlannerSourceTree(t, map[string]string{"Alpha.txt": "a"})
	secondSource := makePlannerSourceTree(t, map[string]string{"Beta.txt": "b"})
	firstModID := insertPlannerMod(t, store, gameID, "Alpha", firstSource)
	secondModID := insertPlannerMod(t, store, gameID, "Beta", secondSource)

	addPlannerProfileMod(t, store, profileID, firstModID, true, 0)
	addPlannerProfileMod(t, store, profileID, secondModID, true, 1)

	addPlannerInstallConfig(t, store, firstModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Shared/Folder", nil)
	addPlannerInstallConfig(t, store, secondModID, string(installconfig.StrategyTypeUnrealPak), installconfig.TargetBaseGameRoot, "Shared/Folder", nil)

	resolved, err := ResolveProfilePlan(context.Background(), store, profileID)
	if err != nil {
		t.Fatalf("ResolveProfilePlan() error = %v", err)
	}

	plan, err := BuildOperationPlan(resolved)
	if err != nil {
		t.Fatalf("BuildOperationPlan() error = %v", err)
	}

	if len(plan.Operations) < 2 {
		t.Fatalf("BuildOperationPlan() operations = %+v, want at least 2", plan.Operations)
	}

	sharedDir := filepath.Join(gameRoot, "Shared", "Folder")
	var dirOperation *Operation
	for index := range plan.Operations {
		if plan.Operations[index].Type == OperationTypeCreateDirectory && plan.Operations[index].TargetPath == sharedDir {
			dirOperation = &plan.Operations[index]
			break
		}
	}
	if dirOperation == nil {
		t.Fatalf("create_directory for %q not found in %+v", sharedDir, plan.Operations)
	}
	if dirOperation.Mod.ModID != firstModID {
		t.Fatalf("shared directory owner = %+v, want first mod %d", dirOperation.Mod, firstModID)
	}
}

func TestBuildOperationPlanReturnsErrorWhenGameModStoragePathIsMissing(t *testing.T) {
	t.Parallel()

	_, err := BuildOperationPlan(ResolveProfilePlanResult{
		GameInstallPath: "/games/skyrim",
		Mods: []ProfilePlanMod{
			{
				ModID:              1,
				ModName:            "SkyUI",
				ManagedSourcePath:  "/managed/skyui",
				StrategyType:       installconfig.StrategyTypeGenericCopy,
				TargetBase:         installconfig.TargetBaseGameRoot,
				TargetRelativePath: "Data",
			},
		},
	})
	if err == nil {
		t.Fatal("BuildOperationPlan() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "build operation plan") || !strings.Contains(err.Error(), "game mod storage path is required") {
		t.Fatalf("BuildOperationPlan() error = %q, want wrapped missing managed storage path error", err.Error())
	}
}

func makePlannerSourceTree(t *testing.T, files map[string]string) string {
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
