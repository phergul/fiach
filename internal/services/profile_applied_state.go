package services

import (
	"context"
	"fmt"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment/execute"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func (s *ProfileService) LoadAppliedFileStates(ctx context.Context, gameID int64) (states []appliedstate.PersistedFileState, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("load applied file states: %w", err)
		}
	}()

	if gameID <= 0 {
		return nil, apperror.New("A valid game must be selected.")
	}

	_, found, err := s.store.GetAppliedProfileState(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if !found {
		return []appliedstate.PersistedFileState{}, nil
	}

	rows, err := s.store.ListAppliedFileStates(ctx, gameID)
	if err != nil {
		return nil, err
	}

	return fromDBAppliedFileStateRows(rows), nil
}

func toDBAppliedFileStateRows(gameID int64, states []appliedstate.PersistedFileState, appliedAt string) []dbtypes.AppliedFileStateRow {
	rows := make([]dbtypes.AppliedFileStateRow, len(states))
	for index, state := range states {
		lastAppliedAt := state.LastAppliedAt
		if lastAppliedAt == "" {
			lastAppliedAt = appliedAt
		}

		rows[index] = dbtypes.AppliedFileStateRow{
			GameID:             gameID,
			GameRelativePath:   state.GameRelativePath,
			ProfileID:          state.ProfileID,
			BaselineExists:     state.BaselineExists,
			BaselineSHA256:     state.BaselineSHA256,
			BaselineSizeBytes:  state.BaselineSizeBytes,
			BaselineBackupPath: state.BaselineBackupPath,
			AppliedExists:      state.AppliedExists,
			AppliedSHA256:      state.AppliedSHA256,
			AppliedSizeBytes:   state.AppliedSizeBytes,
			WinningSourceKind:  state.WinningSourceKind,
			WinningSourceID:    state.WinningSourceID,
			WinningModID:       state.WinningModID,
			WinningLoadOrder:   state.WinningLoadOrder,
			OutputKind:         state.OutputKind,
			UserDecision:       state.UserDecision,
			LastAppliedAt:      lastAppliedAt,
		}
	}

	return rows
}

func fromDBAppliedFileStateRows(rows []dbtypes.AppliedFileStateRow) []appliedstate.PersistedFileState {
	states := make([]appliedstate.PersistedFileState, len(rows))
	for index, row := range rows {
		states[index] = appliedstate.PersistedFileState{
			GameID:             row.GameID,
			GameRelativePath:   row.GameRelativePath,
			ProfileID:          row.ProfileID,
			BaselineExists:     row.BaselineExists,
			BaselineSHA256:     row.BaselineSHA256,
			BaselineSizeBytes:  row.BaselineSizeBytes,
			BaselineBackupPath: row.BaselineBackupPath,
			AppliedExists:      row.AppliedExists,
			AppliedSHA256:      row.AppliedSHA256,
			AppliedSizeBytes:   row.AppliedSizeBytes,
			WinningSourceKind:  row.WinningSourceKind,
			WinningSourceID:    row.WinningSourceID,
			WinningModID:       row.WinningModID,
			WinningLoadOrder:   row.WinningLoadOrder,
			OutputKind:         row.OutputKind,
			UserDecision:       row.UserDecision,
			LastAppliedAt:      row.LastAppliedAt,
		}
	}

	return states
}

func toDBAppliedCreatedDirectoryRows(
	gameID int64,
	installPath string,
	outcome execute.FirstApplyOutcome,
) ([]dbtypes.AppliedCreatedDirectoryRow, error) {
	rows := make([]dbtypes.AppliedCreatedDirectoryRow, 0, len(outcome.CreatedDirectories))
	for _, directory := range outcome.CreatedDirectories {
		gameRelativePath, err := appliedstate.AbsoluteToGameRelativePath(installPath, directory.TargetPath)
		if err != nil {
			return nil, fmt.Errorf("created directory %q: %w", directory.TargetPath, err)
		}

		row := dbtypes.AppliedCreatedDirectoryRow{
			GameID:           gameID,
			GameRelativePath: gameRelativePath,
		}
		if directory.ModID > 0 {
			modID := directory.ModID
			row.ModID = &modID
		}
		if directory.ModName != "" {
			modName := directory.ModName
			row.ModName = &modName
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func fromDBAppliedCreatedDirectoryRows(rows []dbtypes.AppliedCreatedDirectoryRow) []execute.RestoreCreatedDirectory {
	directories := make([]execute.RestoreCreatedDirectory, len(rows))
	for index, row := range rows {
		directories[index] = execute.RestoreCreatedDirectory{
			GameRelativePath: row.GameRelativePath,
			ModID:            row.ModID,
			ModName:          row.ModName,
		}
	}

	return directories
}
