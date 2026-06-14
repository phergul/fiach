package filetxn

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type testAction string

func TestRecoveryJournalRemovesCommittedAndReturnsOldestPending(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	journalsDir := filepath.Join(dataDir, "journals")
	if err := WriteJournal(filepath.Join(journalsDir, "001.json"), Journal[testAction]{
		Version: 1, ID: "001", DatabaseCommitted: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := WriteJournal(filepath.Join(journalsDir, "002.json"), Journal[testAction]{
		Version: 1, ID: "002", Action: "install",
	}); err != nil {
		t.Fatal(err)
	}
	journal, required, err := RecoveryJournal[testAction](dataDir, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !required || journal.ID != "002" || journal.Action != "install" {
		t.Fatalf("RecoveryJournal() = %+v, %v", journal, required)
	}
	if _, err := os.Stat(filepath.Join(journalsDir, "001.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("committed journal remains: %v", err)
	}
}

func TestRollbackJournalPersistsFailureAndRemovesSuccess(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	path := JournalPath(dataDir, "journal-1")
	if err := WriteJournal(path, Journal[testAction]{
		Version: 1, ID: "journal-1", Snapshots: []Snapshot{{TargetPath: "target"}},
	}); err != nil {
		t.Fatal(err)
	}
	rollbackErr := errors.New("rollback failed")
	journal, err := RollbackJournal[testAction](dataDir, "journal-1", 1, func([]Snapshot) error {
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) || journal.Error != rollbackErr.Error() {
		t.Fatalf("RollbackJournal(failure) = %+v, %v", journal, err)
	}
	persisted, err := ReadJournal[testAction](path, 1)
	if err != nil || persisted.Error != rollbackErr.Error() {
		t.Fatalf("ReadJournal() = %+v, %v", persisted, err)
	}
	if _, err := RollbackJournal[testAction](dataDir, "journal-1", 1, func([]Snapshot) error {
		return nil
	}); err != nil {
		t.Fatalf("RollbackJournal(success) error = %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("successful journal remains: %v", err)
	}
}

func TestReadJournalRejectsUnsupportedVersion(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "journal.json")
	if err := WriteJournal(path, Journal[testAction]{Version: 2, ID: "journal"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadJournal[testAction](path, 1); err == nil {
		t.Fatal("ReadJournal() error = nil")
	}
}
