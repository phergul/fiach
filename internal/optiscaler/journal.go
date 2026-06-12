package optiscaler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/phergul/fiach/internal/fileops"
)

type journalSnapshot struct {
	TargetPath string `json:"targetPath"`
	Existed    bool   `json:"existed"`
	BackupPath string `json:"backupPath,omitempty"`
	SHA256     string `json:"sha256,omitempty"`
	SizeBytes  int64  `json:"sizeBytes,omitempty"`
}

type journalDocument struct {
	Version            int               `json:"version"`
	ID                 string            `json:"id"`
	GameID             int64             `json:"gameId"`
	TargetPath         string            `json:"targetPath"`
	TargetRelativePath string            `json:"targetRelativePath"`
	Action             Action            `json:"action"`
	StartedAt          time.Time         `json:"startedAt"`
	CompletedSteps     int               `json:"completedSteps"`
	DatabaseCommitted  bool              `json:"databaseCommitted"`
	Snapshots          []journalSnapshot `json:"snapshots"`
	Error              string            `json:"error,omitempty"`
}

func (m *Manager) RecoveryState() (state RecoveryState, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("read OptiScaler recovery state: %w", err)
		}
	}()

	journals, err := m.pendingJournals()
	if err != nil {
		return RecoveryState{}, err
	}
	if len(journals) == 0 {
		return RecoveryState{}, nil
	}
	journal, err := readJournal(journals[0])
	if err != nil {
		return RecoveryState{}, err
	}
	if journal.DatabaseCommitted {
		if err := os.Remove(journals[0]); err != nil {
			return RecoveryState{}, err
		}
		_ = os.RemoveAll(filepath.Join(m.dataDir, "journals", journal.ID))
		return m.RecoveryState()
	}
	return RecoveryState{
		Required: true, JournalID: journal.ID, GameID: journal.GameID,
		TargetPath: journal.TargetPath, Action: journal.Action,
		StartedAt: journal.StartedAt, Error: journal.Error,
	}, nil
}

func (m *Manager) RollbackRecovery(journalID string) (result ApplyResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("rollback OptiScaler recovery journal: %w", err)
		}
	}()
	m.mu.Lock()
	defer m.mu.Unlock()

	path := filepath.Join(m.dataDir, "journals", safePathSegment(journalID)+".json")
	journal, err := readJournal(path)
	if err != nil {
		return ApplyResult{}, err
	}
	if journal.ID != journalID {
		return ApplyResult{}, errors.New("journal ID does not match")
	}
	if err := rollbackSnapshots(journal.Snapshots); err != nil {
		journal.Error = err.Error()
		_ = writeJournal(path, journal)
		return ApplyResult{Success: false, Message: err.Error()}, err
	}
	if err := os.Remove(path); err != nil {
		return ApplyResult{}, err
	}
	_ = os.RemoveAll(filepath.Join(m.dataDir, "journals", journal.ID))
	if target, found, targetErr := m.store.GetOptiScalerTarget(context.Background(), journal.GameID, journal.TargetRelativePath); targetErr != nil {
		return ApplyResult{}, targetErr
	} else if found && target.Status == "recovery_required" {
		if targetErr := m.saveTargetStatus(context.Background(), target, "managed"); targetErr != nil {
			return ApplyResult{}, targetErr
		}
	}
	return ApplyResult{Success: true, RolledBack: true, Message: "Recovery rollback completed."}, nil
}

func (m *Manager) pendingJournals() ([]string, error) {
	root := filepath.Join(m.dataDir, "journals")
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			paths = append(paths, filepath.Join(root, entry.Name()))
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func readJournal(path string) (journalDocument, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return journalDocument{}, err
	}
	var journal journalDocument
	if err := json.Unmarshal(contents, &journal); err != nil {
		return journalDocument{}, err
	}
	if journal.Version != JournalVersion {
		return journalDocument{}, fmt.Errorf("unsupported journal version %d", journal.Version)
	}
	return journal, nil
}

func writeJournal(path string, journal journalDocument) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".journal-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(contents); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(tempPath, path)
}

func rollbackSnapshots(snapshots []journalSnapshot) error {
	for index := len(snapshots) - 1; index >= 0; index-- {
		snapshot := snapshots[index]
		if snapshot.Existed {
			if err := os.MkdirAll(filepath.Dir(snapshot.TargetPath), 0o755); err != nil {
				return err
			}
			if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
				SourcePath: snapshot.BackupPath, TargetPath: snapshot.TargetPath,
				Mode: 0o644, Replace: true, OpenLabel: "journal backup",
			}); err != nil {
				return err
			}
			matches, err := fileops.FileMatchesIntegrity(snapshot.TargetPath, snapshot.SHA256, snapshot.SizeBytes)
			if err != nil || !matches {
				return fmt.Errorf("restored file %q failed verification", snapshot.TargetPath)
			}
		} else if err := os.Remove(snapshot.TargetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
