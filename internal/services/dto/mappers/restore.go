package mappers

import (
	"github.com/phergul/fiach/internal/deployment/execute"
	"github.com/phergul/fiach/internal/services/dto"
)

func ToDTORestoreResult(result execute.VanillaRestoreResult) dto.RestoreResult {
	return dto.RestoreResult{
		Success:        result.Success,
		CompletedCount: result.CompletedCount,
		FailedCount:    result.FailedCount,
		SkippedCount:   result.SkippedCount,
		Results:        ToDTORestoreOperationResults(result.Results),
	}
}

func ToDTORestoreOperationResults(results []execute.VanillaRestoreOperationResult) []dto.RestoreOperationResult {
	dtoResults := make([]dto.RestoreOperationResult, 0, len(results))
	for _, result := range results {
		dtoResults = append(dtoResults, dto.RestoreOperationResult{
			OperationIndex: result.OperationIndex,
			Operation:      ToDTORestoreOperation(result.Operation),
			Status:         dto.RestoreOperationStatus(result.Status),
			Message:        result.Message,
			Error:          result.Error,
		})
	}
	return dtoResults
}

func ToDTORestoreOperation(operation execute.VanillaRestoreOperation) dto.RestoreOperation {
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
