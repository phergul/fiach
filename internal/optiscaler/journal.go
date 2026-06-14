package optiscaler

import (
	"context"
	"fmt"

	"github.com/phergul/fiach/internal/filetxn"
)

type journalSnapshot = filetxn.Snapshot
type journalDocument = filetxn.Journal[Action]

func (m *Manager) RecoveryState() (state RecoveryState, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("read OptiScaler recovery state: %w", err)
		}
	}()

	journal, required, err := filetxn.RecoveryJournal[Action](m.dataDir, JournalVersion)
	if err != nil {
		return RecoveryState{}, err
	}
	if !required {
		return RecoveryState{}, nil
	}
	return RecoveryState{
		Required:   true,
		JournalID:  journal.ID,
		GameID:     journal.GameID,
		TargetPath: journal.TargetPath,
		Action:     journal.Action,
		StartedAt:  journal.StartedAt,
		Error:      journal.Error,
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

	journal, err := filetxn.RollbackJournal[Action](
		m.dataDir, journalID, JournalVersion, rollbackSnapshots)
	if err != nil {
		return ApplyResult{
			Success: false,
			Message: err.Error(),
		}, err
	}
	if target, found, targetErr := m.store.GetOptiScalerTarget(context.Background(), journal.GameID, journal.TargetRelativePath); targetErr != nil {
		return ApplyResult{}, targetErr
	} else if found && target.Status == "recovery_required" {
		if targetErr := m.saveTargetStatus(context.Background(), target, "managed"); targetErr != nil {
			return ApplyResult{}, targetErr
		}
	}
	return ApplyResult{
		Success:    true,
		RolledBack: true,
		Message:    "Recovery rollback completed.",
	}, nil
}

func writeJournal(path string, journal journalDocument) error {
	return filetxn.WriteJournal(path, journal)
}

func rollbackSnapshots(snapshots []journalSnapshot) error {
	return filetxn.RollbackSnapshots(snapshots)
}
