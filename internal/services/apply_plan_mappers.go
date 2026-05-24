package services

import (
	"github.com/phergul/mod-manager/internal/operationplan"
	"github.com/phergul/mod-manager/internal/services/dto"
)

func toDTOApplyOperationPlanResult(result operationplan.ApplyOperationPlanResult) dto.ApplyOperationPlanResult {
	return dto.ApplyOperationPlanResult{
		Success:        result.Success,
		CompletedCount: result.CompletedCount,
		FailedCount:    result.FailedCount,
		SkippedCount:   result.SkippedCount,
		Results:        toDTOApplyOperationResults(result.Results),
		Manifest:       toDTOAppliedOperationManifest(result.Manifest),
	}
}

func toDTOApplyOperationResults(results []operationplan.ApplyOperationResult) []dto.ApplyOperationResult {
	dtoResults := make([]dto.ApplyOperationResult, 0, len(results))
	for _, result := range results {
		dtoResults = append(dtoResults, dto.ApplyOperationResult{
			OperationIndex: result.OperationIndex,
			Operation:      toDTOOperation(result.Operation),
			Status:         dto.ApplyOperationStatus(result.Status),
			Message:        result.Message,
			Error:          result.Error,
		})
	}
	return dtoResults
}

func toDTOAppliedOperationManifest(manifest operationplan.AppliedOperationManifest) dto.AppliedOperationManifest {
	return dto.AppliedOperationManifest{
		AddedFiles:         toDTOAppliedFileManifestEntries(manifest.AddedFiles),
		ReplacedFiles:      toDTOReplacedFileManifestEntries(manifest.ReplacedFiles),
		CreatedDirectories: toDTOAppliedDirectoryManifestEntries(manifest.CreatedDirectories),
	}
}

func toDTOAppliedFileManifestEntries(entries []operationplan.AppliedFileManifestEntry) []dto.AppliedFileManifestEntry {
	result := make([]dto.AppliedFileManifestEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, dto.AppliedFileManifestEntry{
			OperationIndex: entry.OperationIndex,
			Mod:            toDTOModContext(entry.Mod),
			SourcePath:     entry.SourcePath,
			TargetPath:     entry.TargetPath,
			SHA256:         entry.SHA256,
			SizeBytes:      entry.SizeBytes,
		})
	}
	return result
}

func toDTOReplacedFileManifestEntries(entries []operationplan.ReplacedFileManifestEntry) []dto.ReplacedFileManifestEntry {
	result := make([]dto.ReplacedFileManifestEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, dto.ReplacedFileManifestEntry{
			OperationIndex:  entry.OperationIndex,
			Mod:             toDTOModContext(entry.Mod),
			SourcePath:      entry.SourcePath,
			TargetPath:      entry.TargetPath,
			SHA256:          entry.SHA256,
			SizeBytes:       entry.SizeBytes,
			BackupPath:      entry.BackupPath,
			BackupSHA256:    entry.BackupSHA256,
			BackupSizeBytes: entry.BackupSizeBytes,
		})
	}
	return result
}

func toDTOAppliedDirectoryManifestEntries(entries []operationplan.AppliedDirectoryManifestEntry) []dto.AppliedDirectoryManifestEntry {
	result := make([]dto.AppliedDirectoryManifestEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, dto.AppliedDirectoryManifestEntry{
			OperationIndex: entry.OperationIndex,
			Mod:            toDTOModContext(entry.Mod),
			TargetPath:     entry.TargetPath,
		})
	}
	return result
}
