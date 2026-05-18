package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	plan, err := service.BuildProfileOperationPlan(profileID)
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
	plan, err := service.BuildProfileOperationPlan(profileID)
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

func TestProfileServiceReturnsStorageConfigurationErrorForOperationPlan(t *testing.T) {
	t.Parallel()

	service := NewProfileService(nil)

	_, err := service.BuildProfileOperationPlan(1)
	if err == nil {
		t.Fatal("BuildProfileOperationPlan() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "build profile operation plan") || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("BuildProfileOperationPlan() error = %q, want service context", err.Error())
	}
}

func TestProfileServiceWrapsUnexpectedPlannerErrors(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewProfileService(store)
	_, err := service.BuildProfileOperationPlan(999)
	if err == nil {
		t.Fatal("BuildProfileOperationPlan() error = nil, want planner error")
	}
	if !strings.Contains(err.Error(), "build profile operation plan") || !strings.Contains(err.Error(), "resolve profile plan") || !strings.Contains(err.Error(), "profile 999 was not found") {
		t.Fatalf("BuildProfileOperationPlan() error = %q, want wrapped resolver detail", err.Error())
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
