package planner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
)

func TestPlanFirstApply_CreateReplaceAndBlock(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	existingPath := filepath.Join(gameRoot, "Data", "existing.esp")
	if err := os.MkdirAll(filepath.Dir(existingPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(existingPath, []byte("vanilla"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	state := deployment.DesiredState{
		Files: map[string]deployment.DesiredFile{
			deployment.CanonicalGameRelativePath("Data/new.esp"): {
				GameRelativePath: "Data/new.esp",
				SHA256:           "desired-new",
				SizeBytes:        8,
				FileStatus:       deployment.FileStatusAdded,
			},
			deployment.CanonicalGameRelativePath("Data/existing.esp"): {
				GameRelativePath: "Data/existing.esp",
				SHA256:           "desired-existing",
				SizeBytes:        6,
				FileStatus:       deployment.FileStatusReplaced,
			},
			deployment.CanonicalGameRelativePath("blocked/file.txt"): {
				GameRelativePath: "blocked/file.txt",
				SHA256:           "desired-blocked",
				SizeBytes:        7,
				FileStatus:       deployment.FileStatusBlocked,
				RiskLevel:        deployment.RiskError,
			},
		},
	}

	plan, err := planner.PlanFirstApply(state, gameRoot)
	if err != nil {
		t.Fatalf("PlanFirstApply() error = %v", err)
	}

	newPlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/new.esp")]
	if newPlan.PlannedAction != planner.ReapplyCreate || newPlan.Current.Exists {
		t.Fatalf("new file plan = %+v, want create with missing current", newPlan)
	}
	if !newPlan.Desired.Exists || newPlan.Desired.SHA256 != "desired-new" {
		t.Fatalf("new file desired = %+v, want desired snapshot", newPlan.Desired)
	}

	replacePlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/existing.esp")]
	if replacePlan.PlannedAction != planner.ReapplyReplace || !replacePlan.Current.Exists || replacePlan.Current.SHA256 == "" {
		t.Fatalf("replace file plan = %+v, want replace with current hash", replacePlan)
	}

	blockedPlan := plan.Paths[deployment.CanonicalGameRelativePath("blocked/file.txt")]
	if blockedPlan.PlannedAction != planner.ReapplyBlock {
		t.Fatalf("blocked file plan = %+v, want block", blockedPlan)
	}

	if plan.CanApply() {
		t.Fatal("PlanFirstApply() CanApply = true, want false for blocked path")
	}
}
