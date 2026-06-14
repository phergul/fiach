package filetxn

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotExecuteVerifyAndRollback(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	journalRoot := t.TempDir()
	source := filepath.Join(t.TempDir(), "source.dll")
	target := filepath.Join(root, "dxgi.dll")
	if err := os.WriteFile(source, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	operation := Operation{Type: "copy", SourcePath: source, TargetPath: target}
	if err := ValidateOperations([]Operation{operation}, root); err != nil {
		t.Fatalf("ValidateOperations() error = %v", err)
	}
	snapshots, err := SnapshotOperations(journalRoot, []Operation{operation})
	if err != nil {
		t.Fatalf("SnapshotOperations() error = %v", err)
	}
	if err := ExecuteOperation(operation, "test source"); err != nil {
		t.Fatalf("ExecuteOperation() error = %v", err)
	}
	if err := RollbackSnapshots(snapshots); err != nil {
		t.Fatalf("RollbackSnapshots() error = %v", err)
	}
	contents, err := os.ReadFile(target)
	if err != nil || string(contents) != "old" {
		t.Fatalf("target after rollback = %q, %v", contents, err)
	}
}

func TestValidateOperationsRejectsEscapingTarget(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	err := ValidateOperations([]Operation{{
		Type: "delete", TargetPath: filepath.Join(root, "..", "outside.dll"),
	}}, root)
	if err == nil {
		t.Fatal("ValidateOperations() error = nil")
	}
}

func TestRollbackFailureIsReturned(t *testing.T) {
	t.Parallel()
	err := RollbackSnapshots([]Snapshot{{
		TargetPath: filepath.Join(t.TempDir(), "target.dll"),
		Existed:    true, BackupPath: filepath.Join(t.TempDir(), "missing.bak"),
	}})
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("RollbackSnapshots() error = %v", err)
	}
}
