package execute_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/execute"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/fileops"
)

func TestBuildOperationsSkipsNoopPaths(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	plan := planner.DeploymentPlan{
		Mode: planner.PlanModeIncremental,
		Paths: map[string]planner.PathPlan{
			deployment.CanonicalGameRelativePath("Data/unchanged.esp"): {
				GameRelativePath: "Data/unchanged.esp",
				PlannedAction:    planner.ReapplyNoOp,
			},
		},
	}

	operations, skippedCount, err := execute.BuildOperations(plan, deployment.DesiredState{}, gameRoot)
	if err != nil {
		t.Fatalf("BuildOperations() error = %v", err)
	}
	if len(operations) != 0 {
		t.Fatalf("operations = %+v, want empty", operations)
	}
	if skippedCount != 1 {
		t.Fatalf("skippedCount = %d, want 1", skippedCount)
	}
}

func TestBuildOperationsCreateReplaceDeleteRestoreRepair(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	modRoot := t.TempDir()
	sourcePath := filepath.Join(modRoot, "plugin.esp")
	if err := os.WriteFile(sourcePath, []byte("desired-content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	desiredSHA256, desiredSize, err := fileops.FileIntegrity(sourcePath)
	if err != nil {
		t.Fatalf("FileIntegrity() error = %v", err)
	}

	targetPath := filepath.Join(gameRoot, "Data", "plugin.esp")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("desired-content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	backupRoot := t.TempDir()
	backupPath := filepath.Join(backupRoot, "deployment-backups", "Data", "plugin.esp")
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(backupPath, []byte("vanilla"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	canonicalPath := deployment.CanonicalGameRelativePath("Data/plugin.esp")
	desired := deployment.DesiredState{
		Files: map[string]deployment.DesiredFile{
			canonicalPath: {
				GameRelativePath: "Data/plugin.esp",
				SourcePath:       sourcePath,
				SHA256:           desiredSHA256,
				SizeBytes:        desiredSize,
			},
		},
	}

	plan := planner.DeploymentPlan{
		Mode: planner.PlanModeIncremental,
		Paths: map[string]planner.PathPlan{
			canonicalPath: {
				GameRelativePath: "Data/plugin.esp",
				PlannedAction:    planner.ReapplyCreate,
			},
			deployment.CanonicalGameRelativePath("Data/remove.esp"): {
				GameRelativePath: "Data/remove.esp",
				PlannedAction:    planner.ReapplyDelete,
			},
			deployment.CanonicalGameRelativePath("Data/restore.esp"): {
				GameRelativePath:   "Data/restore.esp",
				PlannedAction:      planner.ReapplyRestoreBaseline,
				BaselineBackupPath: backupPath,
				Baseline: planner.FileStateSnapshot{
					Exists:    true,
					SHA256:    "0000000000000000000000000000000000000000000000000000000000000001",
					SizeBytes: 7,
				},
			},
			deployment.CanonicalGameRelativePath("Data/repair.esp"): {
				GameRelativePath: "Data/repair.esp",
				PlannedAction:    planner.ReapplyRepair,
			},
		},
	}
	desired.Files[deployment.CanonicalGameRelativePath("Data/repair.esp")] = deployment.DesiredFile{
		GameRelativePath: "Data/repair.esp",
		SHA256:           desiredSHA256,
		SizeBytes:        desiredSize,
	}

	operations, _, err := execute.BuildOperations(plan, desired, gameRoot)
	if err != nil {
		t.Fatalf("BuildOperations() error = %v", err)
	}
	if len(operations) != 4 {
		t.Fatalf("operations = %+v, want 4 operations", operations)
	}

	types := map[string]string{}
	for _, operation := range operations {
		types[filepath.Base(operation.TargetPath)] = operation.Type
	}
	if types["plugin.esp"] != "copy" || types["remove.esp"] != "delete" || types["restore.esp"] != "restore" || types["repair.esp"] != "adopt" {
		t.Fatalf("operation types = %+v, want copy/delete/restore/adopt", types)
	}
}
