package execute_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/drift"
	"github.com/phergul/fiach/internal/deployment/execute"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/fileops"
)

type capturingSaver struct {
	merged []appliedstate.PersistedFileState
}

func (s *capturingSaver) SaveIncrementalAppliedProfileState(
	ctx context.Context,
	gameID int64,
	profileID int64,
	installPath string,
	plan planner.DeploymentPlan,
	desired deployment.DesiredState,
	existingStates []appliedstate.PersistedFileState,
) error {
	merged, err := execute.MergeAppliedFileStates(plan, desired, existingStates, profileID)
	if err != nil {
		return err
	}
	s.merged = merged
	return nil
}

func TestExecuteBackupAndApplyArchivesExternalFileAndClearsDecision(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	sourceRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "Data", "plugin.esp")
	sourcePath := filepath.Join(sourceRoot, "plugin.esp")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("external-edit"), 0o644); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte("desired-content"), 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}

	desiredSHA256, desiredSize, err := fileops.FileIntegrity(sourcePath)
	if err != nil {
		t.Fatalf("FileIntegrity() error = %v", err)
	}

	appliedSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
	appliedSize := int64(1)
	decision := drift.UserDecisionBackupAndApply
	canonicalPath := deployment.CanonicalGameRelativePath("Data/plugin.esp")
	plan := planner.DeploymentPlan{
		Mode: planner.PlanModeIncremental,
		Paths: map[string]planner.PathPlan{
			canonicalPath: {
				GameRelativePath:     "Data/plugin.esp",
				PlannedAction:        planner.ReapplyBackupThenReplace,
				RequiresDriftArchive: true,
				Current: planner.FileStateSnapshot{
					Exists: true,
				},
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
		Desired: deployment.DesiredState{
			Files: map[string]deployment.DesiredFile{
				canonicalPath: {
					GameRelativePath: "Data/plugin.esp",
					SourcePath:       sourcePath,
					SHA256:           desiredSHA256,
					SizeBytes:        desiredSize,
				},
			},
		},
		AppliedFileStates: []appliedstate.PersistedFileState{
			{
				GameRelativePath: "Data/plugin.esp",
				AppliedExists:    true,
				AppliedSHA256:    &appliedSHA256,
				AppliedSizeBytes: &appliedSize,
				UserDecision:     &decision,
			},
		},
		Now: func() time.Time { return time.Unix(100, 0) },
	}, saver)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Fatalf("result = %+v, want success", result)
	}

	archiveRoot := filepath.Join(storageRoot, "archives", "drift", "1", "100000000000")
	archivedPath := filepath.Join(archiveRoot, "Data", "plugin.esp")
	if _, statErr := os.Stat(archivedPath); statErr != nil {
		t.Fatalf("archived file stat = %v, want archived external content", statErr)
	}
	archivedContent, err := os.ReadFile(archivedPath)
	if err != nil {
		t.Fatalf("ReadFile(archive) error = %v", err)
	}
	if string(archivedContent) != "external-edit" {
		t.Fatalf("archived content = %q, want external-edit", string(archivedContent))
	}

	appliedContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile(target) error = %v", err)
	}
	if string(appliedContent) != "desired-content" {
		t.Fatalf("target content = %q, want desired-content", string(appliedContent))
	}

	if len(saver.merged) != 1 {
		t.Fatalf("merged states = %d, want 1", len(saver.merged))
	}
	if saver.merged[0].UserDecision != nil {
		t.Fatalf("UserDecision = %+v, want cleared after backup_and_apply", saver.merged[0].UserDecision)
	}
}

func TestExecuteBackupAndApplyDeleteArchivesBeforeDelete(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "Data", "plugin.esp")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("external-edit"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	canonicalPath := deployment.CanonicalGameRelativePath("Data/plugin.esp")
	plan := planner.DeploymentPlan{
		Mode: planner.PlanModeIncremental,
		Paths: map[string]planner.PathPlan{
			canonicalPath: {
				GameRelativePath:     "Data/plugin.esp",
				PlannedAction:        planner.ReapplyBackupThenDelete,
				RequiresDriftArchive: true,
				Current: planner.FileStateSnapshot{
					Exists: true,
				},
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
		Desired:            deployment.DesiredState{},
		Now:                func() time.Time { return time.Unix(200, 0) },
	}, saver)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Fatalf("result = %+v, want success", result)
	}

	archiveRoot := filepath.Join(storageRoot, "archives", "drift", "1", "200000000000")
	archivedPath := filepath.Join(archiveRoot, "Data", "plugin.esp")
	if _, statErr := os.Stat(archivedPath); statErr != nil {
		t.Fatalf("archived file stat = %v, want archived file", statErr)
	}
	if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) {
		t.Fatalf("target stat = %v, want deleted", statErr)
	}
}
