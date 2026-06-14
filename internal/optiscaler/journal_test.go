package optiscaler

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phergul/fiach/internal/fileops"
)

func TestRecoveryStateAndRollbackRestoreSnapshot(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	target := filepath.Join(t.TempDir(), "dxgi.dll")
	backup := filepath.Join(dataDir, "journals", "journal-1", "000.bak")
	if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
		t.Fatalf("mkdir backup: %v", err)
	}
	if err := os.WriteFile(backup, []byte("original"), 0o644); err != nil {
		t.Fatalf("write backup: %v", err)
	}
	if err := os.WriteFile(target, []byte("changed"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	hash, size, err := fileops.FileIntegrity(backup)
	if err != nil {
		t.Fatalf("FileIntegrity() error = %v", err)
	}
	manager := NewManager(newMemoryStore(), ManagerOptions{DataDir: dataDir})
	journalPath := filepath.Join(dataDir, "journals", "journal-1.json")
	if err := writeJournal(journalPath, journalDocument{
		Version:    JournalVersion,
		ID:         "journal-1",
		GameID:     1,
		TargetPath: filepath.Dir(target),
		Action:     ActionInstall,
		StartedAt:  time.Now(),
		Snapshots: []journalSnapshot{
			{
				TargetPath: target,
				Existed:    true,
				BackupPath: backup,
				SHA256:     hash,
				SizeBytes:  size,
			},
		},
	}); err != nil {
		t.Fatalf("writeJournal() error = %v", err)
	}
	state, err := manager.RecoveryState()
	if err != nil || !state.Required || state.JournalID != "journal-1" {
		t.Fatalf("RecoveryState() = %+v, %v", state, err)
	}
	result, err := manager.RollbackRecovery("journal-1")
	if err != nil || !result.Success || !result.RolledBack {
		t.Fatalf("RollbackRecovery() = %+v, %v", result, err)
	}
	contents, err := os.ReadFile(target)
	if err != nil || string(contents) != "original" {
		t.Fatalf("restored contents = %q, %v", contents, err)
	}
}
