package planner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/drift"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/fileops"
)

func TestPlanIncrementalPreviewDriftedPathRequiresDecision(t *testing.T) {
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
			},
		},
	}

	plan, err := planner.PlanIncrementalPreview(desired, appliedStates, driftResults, gameRoot)
	if err != nil {
		t.Fatalf("PlanIncrementalPreview() error = %v", err)
	}

	pathPlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/plugin.esp")]
	if pathPlan.PlannedAction != planner.ReapplyRequireDecision || pathPlan.FileStatus != deployment.FileStatusDrifted {
		t.Fatalf("path plan = %+v, want drifted require_decision", pathPlan)
	}
	if !pathPlan.Applied.Exists || pathPlan.Baseline.Exists {
		t.Fatalf("applied/baseline = %+v / %+v, want applied populated", pathPlan.Applied, pathPlan.Baseline)
	}
	if plan.CanApply() {
		t.Fatal("PlanIncrementalPreview() CanApply = true, want false")
	}
}

func TestPlanIncrementalPreviewUnchangedAppliedPath(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "Data", "plugin.esp")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("applied-content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appliedSHA256, appliedSize, err := fileops.FileIntegrity(targetPath)
	if err != nil {
		t.Fatalf("FileIntegrity() error = %v", err)
	}

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
				SHA256:           appliedSHA256,
				SizeBytes:        appliedSize,
				FileStatus:       deployment.FileStatusReplaced,
			},
		},
	}

	plan, err := planner.PlanIncrementalPreview(desired, appliedStates, driftResults, gameRoot)
	if err != nil {
		t.Fatalf("PlanIncrementalPreview() error = %v", err)
	}

	pathPlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/plugin.esp")]
	if pathPlan.PlannedAction != planner.ReapplyNoOp || pathPlan.FileStatus != deployment.FileStatusUnchanged {
		t.Fatalf("path plan = %+v, want unchanged noop", pathPlan)
	}
}

func TestPlanIncrementalPreviewExternalDecision(t *testing.T) {
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
	decision := drift.UserDecisionKeepExternal
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
			},
		},
	}

	plan, err := planner.PlanIncrementalPreview(desired, appliedStates, driftResults, gameRoot)
	if err != nil {
		t.Fatalf("PlanIncrementalPreview() error = %v", err)
	}

	pathPlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/plugin.esp")]
	if pathPlan.PlannedAction != planner.ReapplyNoOp || pathPlan.FileStatus != deployment.FileStatusExternal {
		t.Fatalf("path plan = %+v, want external noop", pathPlan)
	}
}
