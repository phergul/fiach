package drift_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/drift"
	"github.com/phergul/fiach/internal/fileops"
)

func TestDetectForPathsNone(t *testing.T) {
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
	applied := []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/plugin.esp",
			AppliedExists:    true,
			AppliedSHA256:    &appliedSHA256,
			AppliedSizeBytes: &appliedSize,
		},
	}

	results, err := drift.DetectForPaths(gameRoot, applied, []string{"Data/plugin.esp"})
	if err != nil {
		t.Fatalf("DetectForPaths() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("DetectForPaths() = %+v, want one result", results)
	}
	if results[0].Kind != deployment.DriftNone {
		t.Fatalf("DetectForPaths() kind = %q, want none", results[0].Kind)
	}
}

func TestDetectForPathsMissing(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	appliedSHA256 := "abc"
	appliedSize := int64(3)
	applied := []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/missing.esp",
			AppliedExists:    true,
			AppliedSHA256:    &appliedSHA256,
			AppliedSizeBytes: &appliedSize,
		},
	}

	results, err := drift.DetectForPaths(gameRoot, applied, []string{"Data/missing.esp"})
	if err != nil {
		t.Fatalf("DetectForPaths() error = %v", err)
	}
	if len(results) != 1 || results[0].Kind != deployment.DriftMissing {
		t.Fatalf("DetectForPaths() = %+v, want missing drift", results)
	}
}

func TestDetectForPathsModified(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "Data", "changed.esp")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("external-edit"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appliedSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
	appliedSize := int64(1)
	applied := []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/changed.esp",
			AppliedExists:    true,
			AppliedSHA256:    &appliedSHA256,
			AppliedSizeBytes: &appliedSize,
		},
	}

	results, err := drift.DetectForPaths(gameRoot, applied, []string{"Data/changed.esp"})
	if err != nil {
		t.Fatalf("DetectForPaths() error = %v", err)
	}
	if len(results) != 1 || results[0].Kind != deployment.DriftModified {
		t.Fatalf("DetectForPaths() = %+v, want modified drift", results)
	}
}

func TestDetectForPathsExternalDecision(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "Data", "external.esp")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("external-edit"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appliedSHA256 := "0000000000000000000000000000000000000000000000000000000000000000"
	appliedSize := int64(1)
	decision := drift.UserDecisionKeepExternal
	applied := []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/external.esp",
			AppliedExists:    true,
			AppliedSHA256:    &appliedSHA256,
			AppliedSizeBytes: &appliedSize,
			UserDecision:     &decision,
		},
	}

	results, err := drift.DetectForPaths(gameRoot, applied, []string{"Data/external.esp"})
	if err != nil {
		t.Fatalf("DetectForPaths() error = %v", err)
	}
	if len(results) != 1 || results[0].Kind != deployment.DriftExternal {
		t.Fatalf("DetectForPaths() = %+v, want external drift", results)
	}
}

func TestDetectForPathsAppliedNeverExisted(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	applied := []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/never.esp",
			AppliedExists:    false,
		},
	}

	results, err := drift.DetectForPaths(gameRoot, applied, []string{"Data/never.esp"})
	if err != nil {
		t.Fatalf("DetectForPaths() error = %v", err)
	}
	if len(results) != 1 || results[0].Kind != deployment.DriftNone {
		t.Fatalf("DetectForPaths() = %+v, want none when applied never existed", results)
	}
}
