package services

import (
	"github.com/phergul/mod-manager/internal/restoreplan"
	"github.com/phergul/mod-manager/internal/services/dto"
)

func toDTORestoreResult(result restoreplan.RestoreResult) dto.RestoreResult {
	return dto.RestoreResult{
		Success:        result.Success,
		CompletedCount: result.CompletedCount,
		FailedCount:    result.FailedCount,
		SkippedCount:   result.SkippedCount,
		Results:        toDTORestoreOperationResults(result.Results),
	}
}

func toDTORestoreOperationResults(results []restoreplan.RestoreOperationResult) []dto.RestoreOperationResult {
	dtoResults := make([]dto.RestoreOperationResult, 0, len(results))
	for _, result := range results {
		dtoResults = append(dtoResults, dto.RestoreOperationResult{
			OperationIndex: result.OperationIndex,
			Operation:      toDTORestoreOperation(result.Operation),
			Status:         dto.RestoreOperationStatus(result.Status),
			Message:        result.Message,
			Error:          result.Error,
		})
	}
	return dtoResults
}

func toDTORestoreOperation(operation restoreplan.RestoreOperation) dto.RestoreOperation {
	return dto.RestoreOperation{
		Type:                   dto.RestoreOperationType(operation.Type),
		ManifestOperationIndex: operation.ManifestOperationIndex,
		Mod: dto.RestoreMod{
			ID:   operation.Mod.ID,
			Name: operation.Mod.Name,
		},
		TargetPath: operation.TargetPath,
		BackupPath: operation.BackupPath,
	}
}
