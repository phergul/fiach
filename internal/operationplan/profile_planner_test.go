package operationplan

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/mod-manager/internal/storage"
)

func TestResolveProfilePlanIncludesEnabledModsInLoadOrder(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	gameID := insertPlannerGame(t, store, "Skyrim", "/games/skyrim")
	profileID := insertPlannerProfile(t, store, gameID, "Default")

	firstModID := insertPlannerMod(t, store, gameID, "First", "/managed/first")
	secondModID := insertPlannerMod(t, store, gameID, "Second", "/managed/second")
	disabledModID := insertPlannerMod(t, store, gameID, "Disabled", "/managed/disabled")

	addPlannerProfileMod(t, store, profileID, secondModID, true, 1)
	addPlannerProfileMod(t, store, profileID, disabledModID, false, 2)
	addPlannerProfileMod(t, store, profileID, firstModID, true, 0)

	addPlannerInstallConfig(t, store, firstModID, "generic_copy", "game_root", "Data", nil)
	sourceSubpath := "plugins/core"
	addPlannerInstallConfig(t, store, secondModID, "replace_files", "game_root", "BepInEx/plugins", &sourceSubpath)
	addPlannerInstallConfig(t, store, disabledModID, "generic_copy", "game_root", "Ignored", nil)

	result, err := ResolveProfilePlan(context.Background(), store, profileID)
	if err != nil {
		t.Fatalf("ResolveProfilePlan() error = %v", err)
	}

	if result.ProfileID != profileID || result.GameID != gameID || result.GameInstallPath != "/games/skyrim" {
		t.Fatalf("ResolveProfilePlan() context = %+v, want profile/game/install path", result)
	}
	if result.GameModStoragePath == "" {
		t.Fatalf("ResolveProfilePlan() GameModStoragePath = empty, want resolved managed storage path")
	}
	if len(result.Issues) != 0 {
		t.Fatalf("ResolveProfilePlan() issues = %+v, want none", result.Issues)
	}
	if len(result.Mods) != 2 {
		t.Fatalf("ResolveProfilePlan() mod count = %d, want 2", len(result.Mods))
	}

	first := result.Mods[0]
	second := result.Mods[1]
	if first.ModID != firstModID || first.ModName != "First" || first.ManagedSourcePath != "/managed/first" || first.LoadOrder != 0 || first.StrategyType != "generic_copy" || first.TargetBase != "game_root" || first.TargetRelativePath != "Data" || first.SourceSubpath != nil {
		t.Fatalf("first input = %+v, want first enabled mod with config", first)
	}
	if second.ModID != secondModID || second.ModName != "Second" || second.ManagedSourcePath != "/managed/second" || second.LoadOrder != 1 || second.StrategyType != "replace_files" || second.TargetBase != "game_root" || second.TargetRelativePath != "BepInEx/plugins" || second.SourceSubpath == nil || *second.SourceSubpath != sourceSubpath {
		t.Fatalf("second input = %+v, want second enabled mod with source subpath", second)
	}
}

func TestResolveProfilePlanReportsMissingInstallConfig(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	gameID := insertPlannerGame(t, store, "Skyrim", "/games/skyrim")
	profileID := insertPlannerProfile(t, store, gameID, "Default")
	modID := insertPlannerMod(t, store, gameID, "SkyUI", "/managed/skyui")
	addPlannerProfileMod(t, store, profileID, modID, true, 0)

	result, err := ResolveProfilePlan(context.Background(), store, profileID)
	if err != nil {
		t.Fatalf("ResolveProfilePlan() error = %v", err)
	}

	if len(result.Mods) != 0 {
		t.Fatalf("ResolveProfilePlan() mods = %+v, want none", result.Mods)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("ResolveProfilePlan() issue count = %d, want 1", len(result.Issues))
	}
	issue := result.Issues[0]
	if issue.Severity != PlanIssueSeverityError || issue.Kind != PlanIssueMissingInstallConfig || issue.ProfileID != profileID || issue.Mod == nil || issue.Mod.ModID != modID || issue.Mod.ModName != "SkyUI" || !strings.Contains(issue.Message, "missing an install configuration") {
		t.Fatalf("issue = %+v, want missing install config issue", issue)
	}
}

func TestResolveProfilePlanReportsIncompleteInstallConfig(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	gameID := insertPlannerGame(t, store, "Skyrim", "/games/skyrim")
	profileID := insertPlannerProfile(t, store, gameID, "Default")
	modID := insertPlannerMod(t, store, gameID, "SkyUI", "/managed/skyui")
	addPlannerProfileMod(t, store, profileID, modID, true, 0)
	addPlannerInstallConfig(t, store, modID, "generic_copy", "game_root", "", nil)

	result, err := ResolveProfilePlan(context.Background(), store, profileID)
	if err != nil {
		t.Fatalf("ResolveProfilePlan() error = %v", err)
	}

	if len(result.Mods) != 0 {
		t.Fatalf("ResolveProfilePlan() mods = %+v, want none", result.Mods)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("ResolveProfilePlan() issue count = %d, want 1", len(result.Issues))
	}
	issue := result.Issues[0]
	if issue.Severity != PlanIssueSeverityError || issue.Kind != PlanIssueIncompleteInstallConfig || issue.Mod == nil || issue.Mod.ModID != modID || !strings.Contains(issue.Message, "incomplete install configuration") {
		t.Fatalf("issue = %+v, want incomplete install config issue", issue)
	}
}

func TestResolveProfilePlanReportsMissingManagedSourcePath(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	gameID := insertPlannerGame(t, store, "Skyrim", "/games/skyrim")
	profileID := insertPlannerProfile(t, store, gameID, "Default")
	modID := insertPlannerMod(t, store, gameID, "SkyUI", "   ")
	addPlannerProfileMod(t, store, profileID, modID, true, 0)
	addPlannerInstallConfig(t, store, modID, "generic_copy", "game_root", "Data", nil)

	result, err := ResolveProfilePlan(context.Background(), store, profileID)
	if err != nil {
		t.Fatalf("ResolveProfilePlan() error = %v", err)
	}

	if len(result.Mods) != 0 {
		t.Fatalf("ResolveProfilePlan() mods = %+v, want none", result.Mods)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("ResolveProfilePlan() issue count = %d, want 1", len(result.Issues))
	}
	issue := result.Issues[0]
	if issue.Severity != PlanIssueSeverityError || issue.Kind != PlanIssueMissingManagedSourcePath || issue.Mod == nil || issue.Mod.ModID != modID || !strings.Contains(issue.Message, "missing a managed source path") {
		t.Fatalf("issue = %+v, want missing managed source path issue", issue)
	}
}

func TestResolveProfilePlanReturnsPartialModsAndAllIssues(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	gameID := insertPlannerGame(t, store, "Skyrim", "/games/skyrim")
	profileID := insertPlannerProfile(t, store, gameID, "Default")

	validModID := insertPlannerMod(t, store, gameID, "Valid", "/managed/valid")
	missingConfigModID := insertPlannerMod(t, store, gameID, "Missing Config", "/managed/missing")
	emptySourceModID := insertPlannerMod(t, store, gameID, "Missing Source", "")

	addPlannerProfileMod(t, store, profileID, validModID, true, 0)
	addPlannerProfileMod(t, store, profileID, missingConfigModID, true, 1)
	addPlannerProfileMod(t, store, profileID, emptySourceModID, true, 2)

	addPlannerInstallConfig(t, store, validModID, "generic_copy", "game_root", "Data", nil)

	beforeProfileMods := countPlannerRows(t, store, "profile_mods")
	beforeConfigs := countPlannerRows(t, store, "mod_install_configs")

	result, err := ResolveProfilePlan(context.Background(), store, profileID)
	if err != nil {
		t.Fatalf("ResolveProfilePlan() error = %v", err)
	}

	if len(result.Mods) != 1 || result.Mods[0].ModID != validModID {
		t.Fatalf("ResolveProfilePlan() mods = %+v, want one valid mod", result.Mods)
	}
	if len(result.Issues) != 2 {
		t.Fatalf("ResolveProfilePlan() issue count = %d, want 2", len(result.Issues))
	}

	afterProfileMods := countPlannerRows(t, store, "profile_mods")
	afterConfigs := countPlannerRows(t, store, "mod_install_configs")
	if afterProfileMods != beforeProfileMods || afterConfigs != beforeConfigs {
		t.Fatalf("row counts changed after resolver: profile_mods %d->%d, mod_install_configs %d->%d", beforeProfileMods, afterProfileMods, beforeConfigs, afterConfigs)
	}
}

func TestResolveProfilePlanReturnsErrorForUnknownProfile(t *testing.T) {
	t.Parallel()

	store := openPlannerStore(t)
	defer closePlannerStore(t, store)

	_, err := ResolveProfilePlan(context.Background(), store, 999)
	if err == nil {
		t.Fatal("ResolveProfilePlan() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "resolve profile plan") || !strings.Contains(err.Error(), "profile 999 was not found") {
		t.Fatalf("ResolveProfilePlan() error = %q, want resolver context and unknown profile detail", err.Error())
	}
}

func TestResolveProfilePlanReturnsStoreConfigurationError(t *testing.T) {
	t.Parallel()

	_, err := ResolveProfilePlan(context.Background(), nil, 1)
	if err == nil {
		t.Fatal("ResolveProfilePlan() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "resolve profile plan") || !strings.Contains(err.Error(), "store is not configured") {
		t.Fatalf("ResolveProfilePlan() error = %q, want resolver context and store configuration detail", err.Error())
	}
}

func openPlannerStore(t *testing.T) *storage.Store {
	t.Helper()

	store, err := storage.Open(context.Background(), storage.Options{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("storage.Open() error = %v", err)
	}
	if err := store.MigrateUp(); err != nil {
		t.Fatalf("store.MigrateUp() error = %v", err)
	}

	return store
}

func closePlannerStore(t *testing.T, store *storage.Store) {
	t.Helper()

	if err := store.Close(); err != nil {
		t.Fatalf("store.Close() error = %v", err)
	}
}

func insertPlannerGame(t *testing.T, store *storage.Store, name string, installPath string) int64 {
	t.Helper()

	result, err := store.DB().Exec(`
		INSERT INTO games (name, install_path)
		VALUES (?, ?)
	`, name, installPath)
	if err != nil {
		t.Fatalf("insert planner game: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("planner game LastInsertId(): %v", err)
	}

	return id
}

func insertPlannerProfile(t *testing.T, store *storage.Store, gameID int64, name string) int64 {
	t.Helper()

	result, err := store.DB().Exec(`
		INSERT INTO profiles (game_id, name)
		VALUES (?, ?)
	`, gameID, name)
	if err != nil {
		t.Fatalf("insert planner profile: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("planner profile LastInsertId(): %v", err)
	}

	return id
}

func insertPlannerMod(t *testing.T, store *storage.Store, gameID int64, name string, sourcePath string) int64 {
	t.Helper()

	originalSourcePath := filepath.Join("/imports", strings.ToLower(strings.ReplaceAll(name, " ", "-")))
	result, err := store.DB().Exec(`
		INSERT INTO mods (game_id, name, source_path, original_source_path)
		VALUES (?, ?, ?, ?)
	`, gameID, name, sourcePath, originalSourcePath)
	if err != nil {
		t.Fatalf("insert planner mod: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("planner mod LastInsertId(): %v", err)
	}

	return id
}

func addPlannerProfileMod(t *testing.T, store *storage.Store, profileID int64, modID int64, enabled bool, loadOrder int64) {
	t.Helper()

	enabledValue := 0
	if enabled {
		enabledValue = 1
	}

	if _, err := store.DB().Exec(`
		INSERT INTO profile_mods (profile_id, mod_id, enabled, load_order)
		VALUES (?, ?, ?, ?)
	`, profileID, modID, enabledValue, loadOrder); err != nil {
		t.Fatalf("insert planner profile mod: %v", err)
	}
}

func addPlannerInstallConfig(t *testing.T, store *storage.Store, modID int64, strategyType string, targetBase string, targetRelativePath string, sourceSubpath *string) {
	t.Helper()

	if _, err := store.DB().Exec(`
		INSERT INTO mod_install_configs (mod_id, strategy_type, target_base, target_relative_path, source_subpath)
		VALUES (?, ?, ?, ?, ?)
	`, modID, strategyType, targetBase, targetRelativePath, sourceSubpath); err != nil {
		t.Fatalf("insert planner install config: %v", err)
	}
}

func countPlannerRows(t *testing.T, store *storage.Store, tableName string) int {
	t.Helper()

	var count int
	if err := store.DB().Get(&count, "SELECT COUNT(*) FROM "+tableName); err != nil {
		t.Fatalf("count rows for %s: %v", tableName, err)
	}

	return count
}
