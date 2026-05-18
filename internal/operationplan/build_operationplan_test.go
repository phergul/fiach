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

	if !plan.CanApply || len(plan.Issues) != 0 {
		t.Fatalf("BuildOperationPlan() plan metadata = %+v, want CanApply=true and no issues", plan)
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

func TestBuildOperationPlanMarksExistingTargetsAsReplaceWithWarningAndManagedBackupPath(t *testing.T) {
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

	if !plan.CanApply {
		t.Fatalf("BuildOperationPlan() CanApply = false, want true for replace warning: %+v", plan)
	}
	if len(plan.Operations) != 1 {
		t.Fatalf("BuildOperationPlan() operation count = %d, want 1", len(plan.Operations))
	}
	if len(plan.Issues) != 1 || plan.Issues[0].Severity != PlanIssueSeverityWarning || plan.Issues[0].Kind != PlanIssueReplaceExistingTarget {
		t.Fatalf("BuildOperationPlan() issues = %+v, want one replace warning", plan.Issues)
	}

	operation := plan.Operations[0]
	wantBackupPath := filepath.Join(resolved.GameModStoragePath, backupRootDirName, "Data", "SkyUI.esp")
	if operation.Type != OperationTypeReplace || operation.Conflict || operation.BackupPath == nil || *operation.BackupPath != wantBackupPath {
		t.Fatalf("replace operation = %+v, want replace without conflict and managed backup path %q", operation, wantBackupPath)
	}
}

func TestBuildOperationPlanReportsMissingGameInstallPathWithoutReturningError(t *testing.T) {
	t.Parallel()

	plan, err := BuildOperationPlan(ResolveProfilePlanResult{})
	if err != nil {
		t.Fatalf("BuildOperationPlan() error = %v, want planner issue result", err)
	}

	if plan.CanApply {
		t.Fatalf("BuildOperationPlan() CanApply = true, want false: %+v", plan)
	}
	if len(plan.Operations) != 0 {
		t.Fatalf("BuildOperationPlan() operations = %+v, want none", plan.Operations)
	}
	if len(plan.Issues) != 1 || plan.Issues[0].Kind != PlanIssueMissingGameInstallPath || plan.Issues[0].Severity != PlanIssueSeverityError {
		t.Fatalf("BuildOperationPlan() issues = %+v, want missing game install path error issue", plan.Issues)
	}
}

func TestBuildOperationPlanReportsMissingGameInstallDirectory(t *testing.T) {
	t.Parallel()

	plan, err := BuildOperationPlan(ResolveProfilePlanResult{
		ProfileID:       1,
		GameInstallPath: filepath.Join(t.TempDir(), "missing"),
	})
	if err != nil {
		t.Fatalf("BuildOperationPlan() error = %v, want planner issue result", err)
	}

	if len(plan.Issues) != 1 || plan.Issues[0].Kind != PlanIssueMissingGameInstallDir {
		t.Fatalf("BuildOperationPlan() issues = %+v, want missing install directory issue", plan.Issues)
	}
}

func TestBuildOperationPlanReportsMissingSourceRootAndSkipsOnlyThatMod(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertPlannerGame(t, store, "Skyrim", gameRoot)
	profileID := insertPlannerProfile(t, store, gameID, "Default")

	validSource := makePlannerSourceTree(t, map[string]string{"Valid.txt": "ok"})
	validModID := insertPlannerMod(t, store, gameID, "Valid", validSource)
	missingModID := insertPlannerMod(t, store, gameID, "Missing", filepath.Join(t.TempDir(), "gone"))

	addPlannerProfileMod(t, store, profileID, validModID, true, 0)
	addPlannerProfileMod(t, store, profileID, missingModID, true, 1)
	addPlannerInstallConfig(t, store, validModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Data", nil)
	addPlannerInstallConfig(t, store, missingModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Data", nil)

	resolved, err := ResolveProfilePlan(context.Background(), store, profileID)
	if err != nil {
		t.Fatalf("ResolveProfilePlan() error = %v", err)
	}

	plan, err := BuildOperationPlan(resolved)
	if err != nil {
		t.Fatalf("BuildOperationPlan() error = %v", err)
	}

	if plan.CanApply {
		t.Fatalf("BuildOperationPlan() CanApply = true, want false for missing source root: %+v", plan)
	}
	if !hasIssueKind(plan.Issues, PlanIssueMissingSourceRoot) {
		t.Fatalf("BuildOperationPlan() issues = %+v, want missing source root issue", plan.Issues)
	}
	if len(plan.Operations) == 0 {
		t.Fatalf("BuildOperationPlan() operations = %+v, want valid mod operations to remain", plan.Operations)
	}
	for _, operation := range plan.Operations {
		if operation.Mod.ModID == missingModID {
			t.Fatalf("BuildOperationPlan() operation = %+v, want missing-source mod skipped", operation)
		}
	}
}

func TestBuildOperationPlanReportsSourceRootThatIsAFile(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertPlannerGame(t, store, "Skyrim", gameRoot)
	profileID := insertPlannerProfile(t, store, gameID, "Default")

	sourceFilePath := filepath.Join(t.TempDir(), "source.txt")
	if err := os.WriteFile(sourceFilePath, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	modID := insertPlannerMod(t, store, gameID, "SkyUI", sourceFilePath)
	addPlannerProfileMod(t, store, profileID, modID, true, 0)
	addPlannerInstallConfig(t, store, modID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Data", nil)

	resolved, err := ResolveProfilePlan(context.Background(), store, profileID)
	if err != nil {
		t.Fatalf("ResolveProfilePlan() error = %v", err)
	}

	plan, err := BuildOperationPlan(resolved)
	if err != nil {
		t.Fatalf("BuildOperationPlan() error = %v", err)
	}

	if !hasIssueKind(plan.Issues, PlanIssueSourceRootNotDirectory) {
		t.Fatalf("BuildOperationPlan() issues = %+v, want source root not directory issue", plan.Issues)
	}
}

func TestBuildOperationPlanReportsExistingFileWhereDirectoryMustBeCreated(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertPlannerGame(t, store, "Skyrim", gameRoot)
	profileID := insertPlannerProfile(t, store, gameID, "Default")

	sourcePath := makePlannerSourceTree(t, map[string]string{
		"Data/SkyUI.esp": "plugin",
	})
	modID := insertPlannerMod(t, store, gameID, "SkyUI", sourcePath)
	addPlannerProfileMod(t, store, profileID, modID, true, 0)
	addPlannerInstallConfig(t, store, modID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Mods/SkyUI", nil)

	blockingPath := filepath.Join(gameRoot, "Mods")
	if err := os.WriteFile(blockingPath, []byte("file"), 0o644); err != nil {
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

	if !hasIssueKind(plan.Issues, PlanIssueTargetDirectoryPathFile) {
		t.Fatalf("BuildOperationPlan() issues = %+v, want target directory path file issue", plan.Issues)
	}
	if len(plan.Operations) != 0 {
		t.Fatalf("BuildOperationPlan() operations = %+v, want mod operations skipped on blocking directory issue", plan.Operations)
	}
}

func TestBuildOperationPlanReportsExistingDirectoryWhereFileMustBeWritten(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertPlannerGame(t, store, "Skyrim", gameRoot)
	profileID := insertPlannerProfile(t, store, gameID, "Default")

	sourcePath := makePlannerSourceTree(t, map[string]string{
		"Data/SkyUI.esp": "plugin",
	})
	modID := insertPlannerMod(t, store, gameID, "SkyUI", sourcePath)
	addPlannerProfileMod(t, store, profileID, modID, true, 0)
	addPlannerInstallConfig(t, store, modID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, ".", nil)

	targetPath := filepath.Join(gameRoot, "Data", "SkyUI.esp")
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	resolved, err := ResolveProfilePlan(context.Background(), store, profileID)
	if err != nil {
		t.Fatalf("ResolveProfilePlan() error = %v", err)
	}

	plan, err := BuildOperationPlan(resolved)
	if err != nil {
		t.Fatalf("BuildOperationPlan() error = %v", err)
	}

	if !hasIssueKind(plan.Issues, PlanIssueTargetFilePathDirectory) {
		t.Fatalf("BuildOperationPlan() issues = %+v, want target file path directory issue", plan.Issues)
	}
}

func TestBuildOperationPlanMarksSameTargetOperationsAsConflicts(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertPlannerGame(t, store, "Skyrim", gameRoot)
	profileID := insertPlannerProfile(t, store, gameID, "Default")

	firstSource := makePlannerSourceTree(t, map[string]string{"plugin.txt": "a"})
	secondSource := makePlannerSourceTree(t, map[string]string{"plugin.txt": "b"})
	firstModID := insertPlannerMod(t, store, gameID, "Alpha", firstSource)
	secondModID := insertPlannerMod(t, store, gameID, "Beta", secondSource)

	addPlannerProfileMod(t, store, profileID, firstModID, true, 0)
	addPlannerProfileMod(t, store, profileID, secondModID, true, 1)
	addPlannerInstallConfig(t, store, firstModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Shared", nil)
	addPlannerInstallConfig(t, store, secondModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Shared", nil)

	resolved, err := ResolveProfilePlan(context.Background(), store, profileID)
	if err != nil {
		t.Fatalf("ResolveProfilePlan() error = %v", err)
	}

	plan, err := BuildOperationPlan(resolved)
	if err != nil {
		t.Fatalf("BuildOperationPlan() error = %v", err)
	}

	if plan.CanApply {
		t.Fatalf("BuildOperationPlan() CanApply = true, want false for target conflict: %+v", plan)
	}
	if !hasIssueKind(plan.Issues, PlanIssueTargetPathConflict) {
		t.Fatalf("BuildOperationPlan() issues = %+v, want target path conflict issue", plan.Issues)
	}

	conflictingCount := 0
	for _, operation := range plan.Operations {
		if operation.Type != OperationTypeCreateDirectory && operation.Conflict {
			conflictingCount++
		}
	}
	if conflictingCount != 2 {
		t.Fatalf("conflicting file operation count = %d, want 2: %+v", conflictingCount, plan.Operations)
	}
}

func TestBuildOperationPlanReportsPartialOperationsAndAccumulatedIssues(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertPlannerGame(t, store, "Skyrim", gameRoot)
	profileID := insertPlannerProfile(t, store, gameID, "Default")

	validSource := makePlannerSourceTree(t, map[string]string{"Valid.txt": "ok"})
	validModID := insertPlannerMod(t, store, gameID, "Valid", validSource)
	missingConfigModID := insertPlannerMod(t, store, gameID, "Missing Config", validSource)
	missingSourceModID := insertPlannerMod(t, store, gameID, "Missing Source", filepath.Join(t.TempDir(), "gone"))

	addPlannerProfileMod(t, store, profileID, validModID, true, 0)
	addPlannerProfileMod(t, store, profileID, missingConfigModID, true, 1)
	addPlannerProfileMod(t, store, profileID, missingSourceModID, true, 2)
	addPlannerInstallConfig(t, store, validModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Data", nil)
	addPlannerInstallConfig(t, store, missingSourceModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Data", nil)

	resolved, err := ResolveProfilePlan(context.Background(), store, profileID)
	if err != nil {
		t.Fatalf("ResolveProfilePlan() error = %v", err)
	}

	plan, err := BuildOperationPlan(resolved)
	if err != nil {
		t.Fatalf("BuildOperationPlan() error = %v", err)
	}

	if len(plan.Operations) == 0 {
		t.Fatalf("BuildOperationPlan() operations = %+v, want valid operations to remain", plan.Operations)
	}
	if !hasIssueKind(plan.Issues, PlanIssueMissingInstallConfig) || !hasIssueKind(plan.Issues, PlanIssueMissingSourceRoot) {
		t.Fatalf("BuildOperationPlan() issues = %+v, want accumulated resolver and builder issues", plan.Issues)
	}
}

func hasIssueKind(issues []PlanIssue, kind PlanIssueKind) bool {
	for _, issue := range issues {
		if issue.Kind == kind {
			return true
		}
	}

	return false
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
