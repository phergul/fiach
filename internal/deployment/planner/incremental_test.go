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

func TestPlanIncrementalRemovedAddedPathDeletes(t *testing.T) {
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
	if pathPlan.PlannedAction != planner.ReapplyDelete || pathPlan.FileStatus != deployment.FileStatusDeleted {
		t.Fatalf("path plan = %+v, want deleted delete", pathPlan)
	}
}

func TestPlanIncrementalRemovedReplacedPathRestoresBaseline(t *testing.T) {
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
	baselineSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
	baselineSize := int64(1)
	backupPath := "/tmp/baseline-backup"

	appliedStates := []appliedstate.PersistedFileState{
		{
			GameRelativePath:   "Data/plugin.esp",
			AppliedExists:      true,
			AppliedSHA256:      &appliedSHA256,
			AppliedSizeBytes:   &appliedSize,
			BaselineExists:     true,
			BaselineSHA256:     &baselineSHA256,
			BaselineSizeBytes:  &baselineSize,
			BaselineBackupPath: &backupPath,
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
	if pathPlan.PlannedAction != planner.ReapplyRestoreBaseline || pathPlan.FileStatus != deployment.FileStatusRestored {
		t.Fatalf("path plan = %+v, want restored restore_baseline", pathPlan)
	}
}

func TestPlanIncrementalRemovedReplacedPathBlocksWithoutBaselineBackup(t *testing.T) {
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
	baselineSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
	baselineSize := int64(1)

	appliedStates := []appliedstate.PersistedFileState{
		{
			GameRelativePath:  "Data/plugin.esp",
			AppliedExists:     true,
			AppliedSHA256:     &appliedSHA256,
			AppliedSizeBytes:  &appliedSize,
			BaselineExists:    true,
			BaselineSHA256:    &baselineSHA256,
			BaselineSizeBytes: &baselineSize,
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
	if pathPlan.PlannedAction != planner.ReapplyBlock || pathPlan.FileStatus != deployment.FileStatusBlocked {
		t.Fatalf("path plan = %+v, want blocked block", pathPlan)
	}
}

func TestPlanIncrementalDesiredChangedDiskMatchesAppliedReplaces(t *testing.T) {
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
	if pathPlan.PlannedAction != planner.ReapplyReplace || pathPlan.FileStatus != deployment.FileStatusReplaced {
		t.Fatalf("path plan = %+v, want replaced replace", pathPlan)
	}
}

func TestPlanIncrementalAllHashesMatchNoops(t *testing.T) {
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
	if pathPlan.PlannedAction != planner.ReapplyNoOp || pathPlan.FileStatus != deployment.FileStatusUnchanged {
		t.Fatalf("path plan = %+v, want unchanged noop", pathPlan)
	}
}

func TestPlanIncrementalDiskDriftRequiresDecision(t *testing.T) {
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
	if pathPlan.PlannedAction != planner.ReapplyRequireDecision || pathPlan.FileStatus != deployment.FileStatusDrifted {
		t.Fatalf("path plan = %+v, want drifted require_decision", pathPlan)
	}
}

func TestPlanIncrementalKeepExternalNoops(t *testing.T) {
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
	if pathPlan.PlannedAction != planner.ReapplyNoOp || pathPlan.FileStatus != deployment.FileStatusExternal {
		t.Fatalf("path plan = %+v, want external noop", pathPlan)
	}
}

func TestPlanIncrementalRepairWhenDiskMatchesDesired(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "Data", "plugin.esp")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("desired-content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	desiredSHA256, desiredSize, err := fileops.FileIntegrity(targetPath)
	if err != nil {
		t.Fatalf("FileIntegrity() error = %v", err)
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
				SHA256:           desiredSHA256,
				SizeBytes:        desiredSize,
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
	if pathPlan.PlannedAction != planner.ReapplyRepair || pathPlan.FileStatus != deployment.FileStatusDrifted {
		t.Fatalf("path plan = %+v, want drifted repair", pathPlan)
	}
}

func TestPlanIncrementalNewPathCreates(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	desired := deployment.DesiredState{
		Files: map[string]deployment.DesiredFile{
			deployment.CanonicalGameRelativePath("Data/new.esp"): {
				GameRelativePath: "Data/new.esp",
				SHA256:           "new-hash",
				SizeBytes:        8,
				FileStatus:       deployment.FileStatusAdded,
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

	plan, err := planner.PlanIncremental(desired, nil, nil, gameRoot)
	if err != nil {
		t.Fatalf("PlanIncremental() error = %v", err)
	}

	pathPlan := plan.Paths[deployment.CanonicalGameRelativePath("Data/new.esp")]
	if pathPlan.PlannedAction != planner.ReapplyCreate || pathPlan.FileStatus != deployment.FileStatusAdded {
		t.Fatalf("path plan = %+v, want added create", pathPlan)
	}
}

func TestPlanIncrementalCanApplyWhenNoBlockers(t *testing.T) {
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

	if !plan.CanApply() {
		t.Fatal("PlanIncremental() CanApply = false, want true without PreviewOnly")
	}
}
