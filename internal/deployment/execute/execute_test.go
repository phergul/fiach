package execute_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/execute"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/fileops"
)

type recordingSaver struct {
	incrementalCalls int
	firstApplyCalls  int
	err              error
}

func (s *recordingSaver) SaveIncrementalAppliedProfileState(
	ctx context.Context,
	gameID int64,
	profileID int64,
	installPath string,
	plan planner.DeploymentPlan,
	desired deployment.DesiredState,
	existingStates []appliedstate.PersistedFileState,
) error {
	s.incrementalCalls++
	return s.err
}

func (s *recordingSaver) SaveFirstApplyAppliedProfileState(
	ctx context.Context,
	gameID int64,
	profileID int64,
	installPath string,
	plan planner.DeploymentPlan,
	desired deployment.DesiredState,
	outcome execute.FirstApplyOutcome,
) error {
	s.firstApplyCalls++
	return s.err
}

func TestExecuteDeletesFileAndRollsBackOnCommitFailure(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "Screenshots", "recording.mov")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("mod-content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	canonicalPath := deployment.CanonicalGameRelativePath("Screenshots/recording.mov")
	plan := planner.DeploymentPlan{
		Mode: planner.PlanModeIncremental,
		Paths: map[string]planner.PathPlan{
			canonicalPath: {
				GameRelativePath: "Screenshots/recording.mov",
				PlannedAction:    planner.ReapplyDelete,
			},
		},
	}

	saver := &recordingSaver{err: errors.New("commit failed")}
	result, err := execute.Execute(context.Background(), execute.Context{
		GameID:             1,
		ProfileID:          2,
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
		PreviewHash:        "preview-hash",
		Plan:               plan,
		Desired:            deployment.DesiredState{},
		Now:                func() time.Time { return time.Unix(0, 0) },
	}, saver)
	if err == nil {
		t.Fatal("Execute() error = nil, want commit failure")
	}
	if !result.RolledBack {
		t.Fatalf("result = %+v, want rolled back", result)
	}
	if _, statErr := os.Stat(targetPath); statErr != nil {
		t.Fatalf("target after rollback stat = %v, want file restored", statErr)
	}
	if saver.incrementalCalls != 1 {
		t.Fatalf("saver calls = %d, want 1 before rollback on commit failure", saver.incrementalCalls)
	}
}

func TestExecuteAppliesDeleteAndCommitsState(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "Screenshots", "recording.mov")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("mod-content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	canonicalPath := deployment.CanonicalGameRelativePath("Screenshots/recording.mov")
	plan := planner.DeploymentPlan{
		Mode: planner.PlanModeIncremental,
		Paths: map[string]planner.PathPlan{
			canonicalPath: {
				GameRelativePath: "Screenshots/recording.mov",
				PlannedAction:    planner.ReapplyDelete,
			},
		},
	}

	saver := &recordingSaver{}
	result, err := execute.Execute(context.Background(), execute.Context{
		GameID:             1,
		ProfileID:          2,
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
		PreviewHash:        "preview-hash",
		Plan:               plan,
		Desired:            deployment.DesiredState{},
		Now:                func() time.Time { return time.Unix(0, 0) },
	}, saver)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || result.CompletedCount != 1 {
		t.Fatalf("result = %+v, want one successful delete", result)
	}
	if _, statErr := os.Stat(targetPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("target after delete stat = %v, want missing file", statErr)
	}
	if saver.incrementalCalls != 1 {
		t.Fatalf("saver calls = %d, want 1", saver.incrementalCalls)
	}
}

func TestMergeAppliedFileStatesRemovesDeletedPath(t *testing.T) {
	t.Parallel()

	canonicalPath := deployment.CanonicalGameRelativePath("Screenshots/recording.mov")
	plan := planner.DeploymentPlan{
		Mode: planner.PlanModeIncremental,
		Paths: map[string]planner.PathPlan{
			canonicalPath: {
				GameRelativePath: "Screenshots/recording.mov",
				PlannedAction:    planner.ReapplyDelete,
			},
		},
	}

	existing := []appliedstate.PersistedFileState{
		{GameRelativePath: "Screenshots/recording.mov", ProfileID: 2, AppliedExists: true},
	}

	merged, err := execute.MergeAppliedFileStates(plan, deployment.DesiredState{}, existing, 2)
	if err != nil {
		t.Fatalf("MergeAppliedFileStates() error = %v", err)
	}
	if len(merged) != 0 {
		t.Fatalf("merged = %+v, want deleted path removed", merged)
	}
}

func TestMergeAppliedFileStatesUpdatesReplacedPath(t *testing.T) {
	t.Parallel()

	modID := int64(9)
	canonicalPath := deployment.CanonicalGameRelativePath("Data/plugin.esp")
	desiredHash := "abc"
	desiredSize := int64(3)
	plan := planner.DeploymentPlan{
		Mode: planner.PlanModeIncremental,
		Paths: map[string]planner.PathPlan{
			canonicalPath: {
				GameRelativePath: "Data/plugin.esp",
				PlannedAction:    planner.ReapplyReplace,
			},
		},
	}
	desired := deployment.DesiredState{
		Files: map[string]deployment.DesiredFile{
			canonicalPath: {
				GameRelativePath: "Data/plugin.esp",
				SHA256:           desiredHash,
				SizeBytes:        desiredSize,
				Winner: deployment.WriterEntry{
					ModID: &modID,
				},
			},
		},
	}

	baselineHash := "baseline"
	existing := []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/plugin.esp",
			ProfileID:        2,
			BaselineExists:   true,
			BaselineSHA256:   &baselineHash,
			AppliedExists:    true,
		},
	}

	merged, err := execute.MergeAppliedFileStates(plan, desired, existing, 2)
	if err != nil {
		t.Fatalf("MergeAppliedFileStates() error = %v", err)
	}
	if len(merged) != 1 {
		t.Fatalf("merged = %+v, want one path", merged)
	}
	if merged[0].AppliedSHA256 == nil || *merged[0].AppliedSHA256 != desiredHash {
		t.Fatalf("merged applied hash = %+v, want %q", merged[0].AppliedSHA256, desiredHash)
	}
	if !merged[0].BaselineExists || merged[0].BaselineSHA256 == nil || *merged[0].BaselineSHA256 != baselineHash {
		t.Fatalf("merged baseline = %+v, want preserved baseline", merged[0])
	}
}

func TestExecuteNoopPlanSkipsFileChanges(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	canonicalPath := deployment.CanonicalGameRelativePath("Data/plugin.esp")
	plan := planner.DeploymentPlan{
		Mode: planner.PlanModeIncremental,
		Paths: map[string]planner.PathPlan{
			canonicalPath: {
				GameRelativePath: "Data/plugin.esp",
				PlannedAction:    planner.ReapplyNoOp,
			},
		},
	}

	saver := &recordingSaver{}
	result, err := execute.Execute(context.Background(), execute.Context{
		GameID:             1,
		ProfileID:          2,
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
		PreviewHash:        "preview-hash",
		Plan:               plan,
		Desired:            deployment.DesiredState{},
	}, saver)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || result.CompletedCount != 0 || result.SkippedCount != 1 {
		t.Fatalf("result = %+v, want noop success", result)
	}
	if saver.incrementalCalls != 0 {
		t.Fatalf("saver calls = %d, want 0 for noop plan", saver.incrementalCalls)
	}
}

func TestBuildOperationsReplaceUsesDesiredSource(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	modRoot := t.TempDir()
	sourcePath := filepath.Join(modRoot, "plugin.esp")
	if err := os.WriteFile(sourcePath, []byte("new"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	sha256, size, err := fileops.FileIntegrity(sourcePath)
	if err != nil {
		t.Fatalf("FileIntegrity() error = %v", err)
	}

	canonicalPath := deployment.CanonicalGameRelativePath("Data/plugin.esp")
	plan := planner.DeploymentPlan{
		Mode: planner.PlanModeIncremental,
		Paths: map[string]planner.PathPlan{
			canonicalPath: {
				GameRelativePath: "Data/plugin.esp",
				PlannedAction:    planner.ReapplyReplace,
			},
		},
	}
	desired := deployment.DesiredState{
		Files: map[string]deployment.DesiredFile{
			canonicalPath: {
				GameRelativePath: "Data/plugin.esp",
				SourcePath:       sourcePath,
				SHA256:           sha256,
				SizeBytes:        size,
			},
		},
	}

	operations, _, err := execute.BuildOperations(plan, desired, gameRoot)
	if err != nil {
		t.Fatalf("BuildOperations() error = %v", err)
	}
	if len(operations) != 1 || operations[0].SourcePath != sourcePath {
		t.Fatalf("operations = %+v, want replace copy from mod source", operations)
	}
}
