package execute_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/execute"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/fileops"
)

func TestExecuteFirstApplyCreatesBaselineBackupAndAppliesReplace(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	modRoot := t.TempDir()
	existingPath := filepath.Join(gameRoot, "Data", "vanilla.txt")
	if err := os.MkdirAll(filepath.Dir(existingPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(existingPath, []byte("vanilla"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	sourcePath := filepath.Join(modRoot, "vanilla.txt")
	if err := os.WriteFile(sourcePath, []byte("modded"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	desiredSHA256, desiredSize, err := fileops.FileIntegrity(sourcePath)
	if err != nil {
		t.Fatalf("FileIntegrity() error = %v", err)
	}

	canonicalPath := deployment.CanonicalGameRelativePath("Data/vanilla.txt")
	plan := planner.DeploymentPlan{
		Mode: planner.PlanModeFirstApply,
		Paths: map[string]planner.PathPlan{
			canonicalPath: {
				GameRelativePath: "Data/vanilla.txt",
				PlannedAction:    planner.ReapplyReplace,
			},
		},
	}
	desired := deployment.DesiredState{
		Files: map[string]deployment.DesiredFile{
			canonicalPath: {
				GameRelativePath: "Data/vanilla.txt",
				SourcePath:       sourcePath,
				SHA256:           desiredSHA256,
				SizeBytes:        desiredSize,
			},
		},
	}

	saver := &capturingSaver{}
	result, err := execute.Execute(context.Background(), execute.Context{
		GameID:             1,
		ProfileID:          2,
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
		PreviewHash:        "preview-hash",
		Plan:               plan,
		Desired:            desired,
		Now:                func() time.Time { return time.Unix(0, 0) },
	}, saver)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || result.CompletedCount != 1 {
		t.Fatalf("Execute() = %+v, want successful replace", result)
	}

	assertServiceFileContents(t, existingPath, "modded")

	backupPath := filepath.Join(storageRoot, "deployment-backups", "Data", "vanilla.txt")
	if _, statErr := os.Stat(backupPath); statErr != nil {
		t.Fatalf("baseline backup stat = %v, want backup file", statErr)
	}
	assertServiceFileContents(t, backupPath, "vanilla")
}

func assertServiceFileContents(t *testing.T, path string, want string) {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	if string(contents) != want {
		t.Fatalf("os.ReadFile(%q) = %q, want %q", path, contents, want)
	}
}
