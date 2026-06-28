package planner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/drift"
	"github.com/phergul/fiach/internal/deployment/planner"
)

func TestPlanIncrementalBackupAndApplyReplacesWithArchive(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "Data", "plugin.esp")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("external-edit"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appliedSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
	appliedSize := int64(1)
	decision := drift.UserDecisionBackupAndApply
	appliedStates := []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/plugin.esp",
			AppliedExists:    true,
			AppliedSHA256:    &appliedSHA256,
			AppliedSizeBytes: &appliedSize,
			UserDecision:     &decision,
		},
	}

	driftResults, err := drift.DetectAll(gameRoot, appliedStates)
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	desired := deployment.DesiredState{
		Files: map[string]deployment.DesiredFile{
			deployment.CanonicalGameRelativePath("Data/plugin.esp"): {
				GameRelativePath: "Data/plugin.esp",
				SHA256:           "desired-hash",
				SizeBytes:        12,
				FileStatus:       deployment.FileStatusReplaced,
				Writers: []deployment.WriterEntry{
					{
						SourceKind: deployment.SourceKindMod,
						ModName:    "Example",
						IsWinner:   true,
					},
				},
			},
		},
	}

	plan, err := planner.PlanIncremental(desired, appliedStates, driftResults, gameRoot)
	if err != nil {
		t.Fatalf("PlanIncremental() error = %v", err)
	}

	pathPlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/plugin.esp")]
	if pathPlan.PlannedAction != planner.ReapplyBackupThenReplace {
		t.Fatalf("path plan action = %q, want backup_then_replace", pathPlan.PlannedAction)
	}
	if !pathPlan.RequiresDriftArchive {
		t.Fatal("RequiresDriftArchive = false, want true")
	}
	if !plan.CanApply() {
		t.Fatal("CanApply() = false, want true after backup_and_apply decision")
	}
}

func TestPlanIncrementalSkippedNoops(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "Data", "plugin.esp")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("external-edit"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appliedSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
	appliedSize := int64(1)
	decision := drift.UserDecisionSkipped
	appliedStates := []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/plugin.esp",
			AppliedExists:    true,
			AppliedSHA256:    &appliedSHA256,
			AppliedSizeBytes: &appliedSize,
			UserDecision:     &decision,
		},
	}

	driftResults, err := drift.DetectAll(gameRoot, appliedStates)
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	desired := deployment.DesiredState{
		Files: map[string]deployment.DesiredFile{
			deployment.CanonicalGameRelativePath("Data/plugin.esp"): {
				GameRelativePath: "Data/plugin.esp",
				SHA256:           "desired-hash",
				SizeBytes:        12,
				FileStatus:       deployment.FileStatusReplaced,
				Writers: []deployment.WriterEntry{
					{
						SourceKind: deployment.SourceKindMod,
						ModName:    "Example",
						IsWinner:   true,
					},
				},
			},
		},
	}

	plan, err := planner.PlanIncremental(desired, appliedStates, driftResults, gameRoot)
	if err != nil {
		t.Fatalf("PlanIncremental() error = %v", err)
	}

	pathPlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/plugin.esp")]
	if pathPlan.PlannedAction != planner.ReapplyNoOp || pathPlan.FileStatus != deployment.FileStatusSkipped {
		t.Fatalf("path plan = %+v, want skipped noop", pathPlan)
	}
	if !plan.CanApply() {
		t.Fatal("CanApply() = false, want true after skipped decision")
	}
}

func TestPlanIncrementalMissingDriftRequiresDecision(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	appliedSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
	appliedSize := int64(1)
	appliedStates := []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/plugin.esp",
			AppliedExists:    true,
			AppliedSHA256:    &appliedSHA256,
			AppliedSizeBytes: &appliedSize,
		},
	}

	driftResults, err := drift.DetectAll(gameRoot, appliedStates)
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	desired := deployment.DesiredState{
		Files: map[string]deployment.DesiredFile{
			deployment.CanonicalGameRelativePath("Data/plugin.esp"): {
				GameRelativePath: "Data/plugin.esp",
				SHA256:           "desired-hash",
				SizeBytes:        12,
				FileStatus:       deployment.FileStatusReplaced,
				Writers: []deployment.WriterEntry{
					{
						SourceKind: deployment.SourceKindMod,
						ModName:    "Example",
						IsWinner:   true,
					},
				},
			},
		},
	}

	plan, err := planner.PlanIncremental(desired, appliedStates, driftResults, gameRoot)
	if err != nil {
		t.Fatalf("PlanIncremental() error = %v", err)
	}

	pathPlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/plugin.esp")]
	if pathPlan.PlannedAction != planner.ReapplyRequireDecision {
		t.Fatalf("path plan action = %q, want require_decision", pathPlan.PlannedAction)
	}
	if pathPlan.DriftKind != deployment.DriftMissing {
		t.Fatalf("DriftKind = %q, want missing", pathPlan.DriftKind)
	}
}

func TestPlanIncrementalMissingDriftApplyDesiredWithoutArchive(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	appliedSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
	appliedSize := int64(1)
	decision := drift.UserDecisionBackupAndApply
	appliedStates := []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/plugin.esp",
			AppliedExists:    true,
			AppliedSHA256:    &appliedSHA256,
			AppliedSizeBytes: &appliedSize,
			UserDecision:     &decision,
		},
	}

	driftResults, err := drift.DetectAll(gameRoot, appliedStates)
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	desired := deployment.DesiredState{
		Files: map[string]deployment.DesiredFile{
			deployment.CanonicalGameRelativePath("Data/plugin.esp"): {
				GameRelativePath: "Data/plugin.esp",
				SHA256:           "desired-hash",
				SizeBytes:        12,
				FileStatus:       deployment.FileStatusReplaced,
				Writers: []deployment.WriterEntry{
					{
						SourceKind: deployment.SourceKindMod,
						ModName:    "Example",
						IsWinner:   true,
					},
				},
			},
		},
	}

	plan, err := planner.PlanIncremental(desired, appliedStates, driftResults, gameRoot)
	if err != nil {
		t.Fatalf("PlanIncremental() error = %v", err)
	}

	pathPlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/plugin.esp")]
	if pathPlan.PlannedAction != planner.ReapplyBackupThenReplace {
		t.Fatalf("path plan action = %q, want backup_then_replace", pathPlan.PlannedAction)
	}
	if pathPlan.RequiresDriftArchive {
		t.Fatal("RequiresDriftArchive = true, want false for missing file")
	}
}

func TestPlanIncrementalRemovedPathDriftRequiresDecision(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "Data", "plugin.esp")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("external-edit"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appliedSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
	appliedSize := int64(1)
	appliedStates := []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/plugin.esp",
			AppliedExists:    true,
			AppliedSHA256:    &appliedSHA256,
			AppliedSizeBytes: &appliedSize,
			BaselineExists:   false,
		},
	}

	driftResults, err := drift.DetectAll(gameRoot, appliedStates)
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	plan, err := planner.PlanIncremental(deployment.DesiredState{Files: map[string]deployment.DesiredFile{}}, appliedStates, driftResults, gameRoot)
	if err != nil {
		t.Fatalf("PlanIncremental() error = %v", err)
	}

	pathPlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/plugin.esp")]
	if pathPlan.PlannedAction != planner.ReapplyRequireDecision {
		t.Fatalf("path plan action = %q, want require_decision", pathPlan.PlannedAction)
	}
}

func TestPlanIncrementalRemovedPathBackupAndApplyDeletesWithArchive(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "Data", "plugin.esp")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("external-edit"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appliedSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
	appliedSize := int64(1)
	decision := drift.UserDecisionBackupAndApply
	appliedStates := []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/plugin.esp",
			AppliedExists:    true,
			AppliedSHA256:    &appliedSHA256,
			AppliedSizeBytes: &appliedSize,
			BaselineExists:   false,
			UserDecision:     &decision,
		},
	}

	driftResults, err := drift.DetectAll(gameRoot, appliedStates)
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	plan, err := planner.PlanIncremental(deployment.DesiredState{Files: map[string]deployment.DesiredFile{}}, appliedStates, driftResults, gameRoot)
	if err != nil {
		t.Fatalf("PlanIncremental() error = %v", err)
	}

	pathPlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/plugin.esp")]
	if pathPlan.PlannedAction != planner.ReapplyBackupThenDelete {
		t.Fatalf("path plan action = %q, want backup_then_delete", pathPlan.PlannedAction)
	}
	if !pathPlan.RequiresDriftArchive {
		t.Fatal("RequiresDriftArchive = false, want true")
	}
}
