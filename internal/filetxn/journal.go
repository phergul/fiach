package filetxn

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Journal[Action ~string] struct {
	Version            int        `json:"version"`
	ID                 string     `json:"id"`
	GameID             int64      `json:"gameId"`
	TargetPath         string     `json:"targetPath"`
	TargetRelativePath string     `json:"targetRelativePath"`
	Action             Action     `json:"action"`
	StartedAt          time.Time  `json:"startedAt"`
	CompletedSteps     int        `json:"completedSteps"`
	DatabaseCommitted  bool       `json:"databaseCommitted"`
	Snapshots          []Snapshot `json:"snapshots"`
	Error              string     `json:"error,omitempty"`
}

func PendingJournals(root string) ([]string, error) {
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

func ReadJournal[Action ~string](path string, supportedVersion int) (Journal[Action], error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Journal[Action]{}, err
	}
	var journal Journal[Action]
	if err := json.Unmarshal(contents, &journal); err != nil {
		return Journal[Action]{}, err
	}
	if journal.Version != supportedVersion {
		return Journal[Action]{}, fmt.Errorf("unsupported journal version %d", journal.Version)
	}
	return journal, nil
}

func WriteJournal[Action ~string](path string, journal Journal[Action]) error {
	return WriteJSONAtomic(path, journal)
}

func JournalPath(dataDir string, journalID string) string {
	return filepath.Join(dataDir, "journals", SafePathSegment(journalID)+".json")
}

func RemoveJournal(dataDir string, path string, journalID string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	_ = os.RemoveAll(filepath.Join(dataDir, "journals", journalID))
	return nil
}

func RecoveryJournal[Action ~string](dataDir string, supportedVersion int) (Journal[Action], bool, error) {
	for {
		paths, err := PendingJournals(filepath.Join(dataDir, "journals"))
		if err != nil {
			return Journal[Action]{}, false, err
		}
		if len(paths) == 0 {
			return Journal[Action]{}, false, nil
		}
		journal, err := ReadJournal[Action](paths[0], supportedVersion)
		if err != nil {
			return Journal[Action]{}, false, err
		}
		if !journal.DatabaseCommitted {
			return journal, true, nil
		}
		if err := RemoveJournal(dataDir, paths[0], journal.ID); err != nil {
			return Journal[Action]{}, false, err
		}
	}
}

func RollbackJournal[Action ~string](
	dataDir string,
	journalID string,
	supportedVersion int,
	rollback func([]Snapshot) error,
) (Journal[Action], error) {
	path := JournalPath(dataDir, journalID)
	journal, err := ReadJournal[Action](path, supportedVersion)
	if err != nil {
		return Journal[Action]{}, err
	}
	if journal.ID != journalID {
		return Journal[Action]{}, errors.New("journal ID does not match")
	}
	if err := rollback(journal.Snapshots); err != nil {
		journal.Error = err.Error()
		_ = WriteJournal(path, journal)
		return journal, err
	}
	if err := RemoveJournal(dataDir, path, journal.ID); err != nil {
		return Journal[Action]{}, err
	}
	return journal, nil
}

func SafePathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "_"
	}
	return strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(value)
}
