package planner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
)

func TestPlanRestorePreviewPlansDeleteRestoreAndBlock(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	addedPath := filepath.Join(gameRoot, "Data", "added.esp")
	replacedPath := filepath.Join(gameRoot, "Data", "replaced.esp")
	if err := os.MkdirAll(filepath.Dir(addedPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(addedPath, []byte("added"), 0o644); err != nil {
		t.Fatalf("WriteFile() added error = %v", err)
	}
	if err := os.WriteFile(replacedPath, []byte("modded"), 0o644); err != nil {
		t.Fatalf("WriteFile() replaced error = %v", err)
	}

	backupPath := filepath.Join(gameRoot, "backups", "replaced.esp")
	baselineSHA256 := "baseline-sha"
	baselineSizeBytes := int64(7)
	appliedSHA256 := "applied-sha"
	appliedSizeBytes := int64(6)
	missingBackupSHA256 := "missing-backup-sha"
	missingBackupSizeBytes := int64(5)
	missingAppliedSHA256 := "missing-applied-sha"
	missingAppliedSizeBytes := int64(4)

	states := []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/added.esp",
			BaselineExists:   false,
			AppliedExists:    true,
			AppliedSHA256:    &appliedSHA256,
			AppliedSizeBytes: &appliedSizeBytes,
			OutputKind:       appliedstate.OutputKindCopied,
		},
		{
			GameRelativePath:   "Data/replaced.esp",
			BaselineExists:     true,
			BaselineSHA256:     &baselineSHA256,
			BaselineSizeBytes:  &baselineSizeBytes,
			BaselineBackupPath: &backupPath,
			AppliedExists:      true,
			AppliedSHA256:      &appliedSHA256,
			AppliedSizeBytes:   &appliedSizeBytes,
			OutputKind:         appliedstate.OutputKindCopied,
		},
		{
			GameRelativePath:  "Data/missing-backup.esp",
			BaselineExists:    true,
			BaselineSHA256:    &missingBackupSHA256,
			BaselineSizeBytes: &missingBackupSizeBytes,
			AppliedExists:     true,
			AppliedSHA256:     &missingAppliedSHA256,
			AppliedSizeBytes:  &missingAppliedSizeBytes,
			OutputKind:        appliedstate.OutputKindCopied,
		},
	}

	plan, err := planner.PlanRestorePreview(states, gameRoot)
	if err != nil {
		t.Fatalf("PlanRestorePreview() error = %v", err)
	}
	if plan.Mode != planner.PlanModeRestorePreview {
		t.Fatalf("PlanRestorePreview() mode = %q, want restore_preview", plan.Mode)
	}

	addedPlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/added.esp")]
	if addedPlan.PlannedAction != planner.ReapplyDelete {
		t.Fatalf("added file plan = %+v, want delete", addedPlan)
	}

	restorePlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/replaced.esp")]
	if restorePlan.PlannedAction != planner.ReapplyRestoreBaseline || restorePlan.BaselineBackupPath != backupPath {
		t.Fatalf("replaced file plan = %+v, want restore baseline", restorePlan)
	}

	blockedPlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/missing-backup.esp")]
	if blockedPlan.PlannedAction != planner.ReapplyBlock {
		t.Fatalf("missing backup plan = %+v, want block", blockedPlan)
	}

	if plan.CanApply() {
		t.Fatal("PlanRestorePreview() CanApply = true, want false for missing backup issue")
	}
	if len(plan.Issues) != 1 || plan.Issues[0].Kind != deployment.PlanIssueMissingBaselineBackup {
		t.Fatalf("PlanRestorePreview() issues = %+v, want missing baseline backup issue", plan.Issues)
	}
}
