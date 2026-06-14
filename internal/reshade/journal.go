package reshade

import (
	"context"
	"fmt"

	"github.com/phergul/fiach/internal/filetxn"
)

type journalDocument = filetxn.Journal[Action]

func (m *Manager) RecoveryState() (state RecoveryState, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("read ReShade recovery state: %w", err)
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
		Required: true, JournalID: journal.ID, GameID: journal.GameID,
		TargetPath: journal.TargetPath, Action: journal.Action,
		StartedAt: journal.StartedAt, Error: journal.Error,
	}, nil
}

func (m *Manager) RollbackRecovery(journalID string) (result ApplyResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("rollback ReShade recovery journal: %w", err)
		}
	}()
	m.mu.Lock()
	defer m.mu.Unlock()

	journal, err := filetxn.RollbackJournal[Action](
		m.dataDir, journalID, JournalVersion, m.rollbackSnapshots)
	if err != nil {
		return ApplyResult{Message: err.Error()}, err
	}
	if row, found, rowErr := m.store.GetReShadeTarget(context.Background(), journal.GameID, journal.TargetRelativePath); rowErr != nil {
		return ApplyResult{}, rowErr
	} else if found && row.Status == "recovery_required" {
		row.Status = "managed"
		_, rowErr = m.store.SaveReShadeTarget(context.Background(), dbInputFromRow(row))
		if rowErr != nil {
			return ApplyResult{}, rowErr
		}
	}
	return ApplyResult{Success: true, RolledBack: true, Message: "ReShade recovery rollback completed."}, nil
}

func writeJournal(path string, journal journalDocument) error {
	return filetxn.WriteJournal(path, journal)
}
