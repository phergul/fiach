package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/installconfig"
	"github.com/phergul/fiach/internal/modimport"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
	"github.com/phergul/fiach/internal/storage/dbtypes"
	"github.com/phergul/fiach/internal/unrealpak"
)

func (s *ModService) PreValidateImport(ctx context.Context, input dto.PreValidateImportInput) (result dto.PreValidateImportResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationPreValidateMod, "Mod import validation started",
		slog.String("source_type", string(input.SourceType)),
		diagnostics.PathAttr("source_path", input.SourcePath),
	)
	defer func() {
		if err != nil {
			diag.fail("Mod import validation failed", err)
			err = fmt.Errorf("pre-validate import: %w", err)
		}
	}()

	source, err := importSource(input.SourceType, input.SourcePath)
	if err != nil {
		return dto.PreValidateImportResult{}, err
	}

	if err := source.Validate(); err != nil {
		return dto.PreValidateImportResult{}, err
	}

	tempPath, err := os.MkdirTemp("", "fiach-import-inspection-*")
	if err != nil {
		return dto.PreValidateImportResult{}, fmt.Errorf("create import inspection folder: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempPath)
	}()

	if err := source.Materialize(tempPath); err != nil {
		return dto.PreValidateImportResult{}, err
	}
	if _, inspectErr := unrealpak.Inspect(tempPath); inspectErr == nil {
		strategy := dto.StrategyTypeUnrealPak
		result.SuggestedStrategy = &strategy
	}

	diag.complete("Mod import validation completed")

	return result, nil
}

func (s *ModService) DetectImportTargets(ctx context.Context, gameID int64, strategyType dto.StrategyType) (result dto.ImportTargetDetectionResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("detect import targets: %w", err)
		}
	}()

	if gameID <= 0 {
		return dto.ImportTargetDetectionResult{}, errors.New("game ID must be positive")
	}
	if strategyType != dto.StrategyTypeUnrealPak {
		return dto.ImportTargetDetectionResult{
			Candidates: []string{},
			Warnings:   []string{},
		}, nil
	}

	game, err := s.store.GetStoredGame(ctx, gameID)
	if err != nil {
		return dto.ImportTargetDetectionResult{}, err
	}
	detection, err := unrealpak.DetectTargets(game.InstallPath)
	if err != nil {
		return dto.ImportTargetDetectionResult{}, err
	}

	return dto.ImportTargetDetectionResult{
		Candidates: detection.Candidates,
		Warnings:   detection.Warnings,
	}, nil
}

func (s *ModService) PreviewImportConfiguration(ctx context.Context, input dto.PreviewImportConfigurationInput) (preview dto.Preview, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationPreviewMod, "Mod import preview started",
		slog.String("source_type", string(input.SourceType)),
		slog.String("strategy_type", string(input.StrategyType)),
		diagnostics.PathAttr("source_path", input.SourcePath),
	)
	defer func() {
		if err != nil {
			diag.fail("Mod import preview failed", err)
			err = fmt.Errorf("preview import configuration: %w", err)
		}
	}()

	source, err := importSource(input.SourceType, input.SourcePath)
	if err != nil {
		return dto.Preview{}, err
	}
	if err := source.Validate(); err != nil {
		return dto.Preview{}, err
	}

	tempPath, err := os.MkdirTemp("", "fiach-import-preview-*")
	if err != nil {
		return dto.Preview{}, fmt.Errorf("create import preview folder: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempPath)
	}()

	if err := source.Materialize(tempPath); err != nil {
		return dto.Preview{}, err
	}

	previewResult, err := installconfig.BuildPreview(installconfig.PreviewInput{
		SourcePath:         tempPath,
		StrategyType:       mappers.ToInstallStrategyType(input.StrategyType),
		TargetRelativePath: input.TargetRelativePath,
		FileCap:            installconfig.DefaultPreviewFileCap,
	})
	if err != nil {
		return dto.Preview{}, err
	}
	targetWarnings, err := s.importTargetWarnings(ctx, input.GameID, input.StrategyType, previewResult.TargetRelativePath)
	if err != nil {
		return dto.Preview{}, err
	}
	previewResult.Warnings = appendUniqueWarnings(previewResult.Warnings, targetWarnings...)

	preview = mappers.ToDTOPreview(previewResult)
	diag.complete("Mod import preview completed",
		slog.Int("file_count", preview.TotalFileCount),
		slog.Int("directory_count", preview.TotalDirectoryCount),
	)

	return preview, nil
}

func (s *ModService) ImportMod(ctx context.Context, input dto.ImportModInput) (result dto.ImportModResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationImportMod, "Mod import started",
		slog.Int64("game_id", input.GameID),
		slog.String("source_type", string(input.SourceType)),
		slog.String("strategy_type", string(input.StrategyType)),
		diagnostics.PathAttr("source_path", input.SourcePath),
	)
	defer func() {
		if err != nil {
			diag.fail("Mod import failed", err)
			err = fmt.Errorf("import mod: %w", err)
		}
	}()

	source, err := importSource(input.SourceType, input.SourcePath)
	if err != nil {
		return dto.ImportModResult{}, err
	}
	targetRelativePath, err := installconfig.NormalizeTargetRelativePath(input.TargetRelativePath)
	if err != nil {
		return dto.ImportModResult{}, err
	}
	targetWarnings, err := s.importTargetWarnings(ctx, input.GameID, input.StrategyType, targetRelativePath)
	if err != nil {
		return dto.ImportModResult{}, err
	}

	importResult, err := modimport.Import(
		ctx,
		s.store,
		input.GameID,
		input.Name,
		source,
		mappers.ToInstallStrategyType(input.StrategyType),
		targetRelativePath,
		modimport.ImportOptions{
			MetadataRegistry: s.metadataRegistry,
		},
	)
	if err != nil {
		return dto.ImportModResult{}, err
	}
	if importResult.MetadataError != nil {
		s.logger.WarnContext(ctx, "Mod metadata unavailable",
			slog.String("operation", diagnostics.OperationImportMod),
			slog.String("event", "metadata_unavailable"),
			slog.Int64("game_id", input.GameID),
			slog.Int64("mod_id", importResult.Mod.ID),
			slog.String("source_type", string(input.SourceType)),
			diagnostics.PathAttr("source_path", input.SourcePath),
			diagnostics.ErrorAttr(importResult.MetadataError),
		)
	}

	diag.complete("Mod import completed",
		slog.Int64("mod_id", importResult.Mod.ID),
	)

	return dto.ImportModResult{
		Mod:      mappers.ToDTOMod(importResult.Mod),
		Config:   mappers.ToDTOModInstallConfig(importResult.Config),
		Warnings: appendUniqueWarnings(targetWarnings, importResult.Warnings...),
	}, nil
}

func (s *ModService) PreviewUpdateMod(ctx context.Context, input dto.UpdateModInput) (result dto.UpdateModResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationPreviewUpdateMod, "Mod update preview started",
		slog.Int64("mod_id", input.ModID),
		slog.String("source_type", string(input.SourceType)),
		diagnostics.PathAttr("source_path", input.SourcePath),
	)
	defer func() {
		if err != nil {
			diag.fail("Mod update preview failed", err)
			err = fmt.Errorf("preview mod update: %w", err)
		}
	}()

	source, err := importSource(input.SourceType, input.SourcePath)
	if err != nil {
		return dto.UpdateModResult{}, err
	}

	updateResult, err := modimport.PreviewUpdate(ctx, s.store, input.ModID, source, modimport.ImportOptions{
		MetadataRegistry: s.metadataRegistry,
	})
	if err != nil {
		return dto.UpdateModResult{}, err
	}

	result, err = s.toUpdateModResult(ctx, updateResult)
	if err != nil {
		return dto.UpdateModResult{}, err
	}
	diag.complete("Mod update preview completed",
		slog.Int64("game_id", updateResult.After.GameID),
		slog.Int64("mod_id", updateResult.After.ID),
		slog.Bool("requires_reapply", result.RequiresReapply),
	)

	return result, nil
}

func (s *ModService) UpdateMod(ctx context.Context, input dto.UpdateModInput) (result dto.UpdateModResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationUpdateMod, "Mod update started",
		slog.Int64("mod_id", input.ModID),
		slog.String("source_type", string(input.SourceType)),
		diagnostics.PathAttr("source_path", input.SourcePath),
	)
	defer func() {
		if err != nil {
			diag.fail("Mod update failed", err)
			err = fmt.Errorf("update mod: %w", err)
		}
	}()

	source, err := importSource(input.SourceType, input.SourcePath)
	if err != nil {
		return dto.UpdateModResult{}, err
	}

	updateResult, err := modimport.Update(ctx, s.store, input.ModID, source, modimport.ImportOptions{
		MetadataRegistry: s.metadataRegistry,
	})
	if err != nil {
		return dto.UpdateModResult{}, err
	}

	result, err = s.toUpdateModResult(ctx, updateResult)
	if err != nil {
		return dto.UpdateModResult{}, err
	}
	if updateResult.MetadataError != nil {
		s.logger.WarnContext(ctx, "Mod metadata unavailable",
			slog.String("operation", diagnostics.OperationUpdateMod),
			slog.String("event", "metadata_unavailable"),
			slog.Int64("game_id", updateResult.After.GameID),
			slog.Int64("mod_id", updateResult.After.ID),
			slog.String("source_type", string(input.SourceType)),
			diagnostics.PathAttr("source_path", input.SourcePath),
			diagnostics.ErrorAttr(updateResult.MetadataError),
		)
	}

	diag.complete("Mod update completed",
		slog.Int64("game_id", updateResult.After.GameID),
		slog.Int64("mod_id", updateResult.After.ID),
		slog.Bool("requires_reapply", result.RequiresReapply),
	)

	return result, nil
}

func (s *ModService) toUpdateModResult(ctx context.Context, updateResult modimport.UpdateResult) (dto.UpdateModResult, error) {
	var metadataWarning *string
	if updateResult.MetadataError != nil {
		warning := updateResult.MetadataError.Error()
		metadataWarning = &warning
	}

	isInAppliedProfile := false
	appliedState, appliedFound, err := s.store.GetAppliedProfileState(ctx, updateResult.After.GameID)
	if err != nil {
		return dto.UpdateModResult{}, err
	}
	if appliedFound {
		isInAppliedProfile, err = s.store.ProfileUsesMod(ctx, appliedState.ProfileID, updateResult.After.ID)
		if err != nil {
			return dto.UpdateModResult{}, err
		}
	}

	return dto.UpdateModResult{
		Mod:                mappers.ToDTOMod(updateResult.After),
		Before:             toModPackageSnapshot(updateResult.Before, updateResult.BeforeMetadata),
		After:              toModPackageSnapshot(updateResult.After, updateResult.AfterMetadata),
		MetadataWarning:    metadataWarning,
		Warnings:           updateResult.Warnings,
		IsInAppliedProfile: isInAppliedProfile,
		RequiresReapply:    isInAppliedProfile,
	}, nil
}

func (s *ModService) importTargetWarnings(ctx context.Context, gameID int64, strategyType dto.StrategyType, targetRelativePath string) ([]string, error) {
	if strategyType != dto.StrategyTypeUnrealPak {
		return []string{}, nil
	}

	detection, err := s.DetectImportTargets(ctx, gameID, strategyType)
	if err != nil {
		return nil, err
	}
	warnings := append([]string{}, detection.Warnings...)
	if !unrealpak.TargetWasDetected(targetRelativePath, detection.Candidates) {
		warnings = append(
			warnings,
			fmt.Sprintf("Target path %q was not detected as an existing Unreal Content/Paks/~mods location.", targetRelativePath),
		)
	}
	return appendUniqueWarnings(nil, warnings...), nil
}

func appendUniqueWarnings(existing []string, warnings ...string) []string {
	result := append([]string{}, existing...)
	seen := make(map[string]struct{}, len(result))
	for _, warning := range result {
		seen[warning] = struct{}{}
	}
	for _, warning := range warnings {
		if _, found := seen[warning]; found {
			continue
		}
		seen[warning] = struct{}{}
		result = append(result, warning)
	}
	return result
}

func importSource(sourceType dto.ModSourceType, sourcePath string) (modimport.Source, error) {
	switch mappers.ToDBModSourceType(sourceType) {
	case dbtypes.ModSourceTypeFolder:
		return modimport.NewFolderSource(sourcePath)
	case dbtypes.ModSourceTypeArchive:
		return modimport.NewArchiveSource(sourcePath)
	case "":
		return nil, errors.New("import source type is required")
	default:
		return nil, fmt.Errorf("unsupported import source type %q", sourceType)
	}
}

func toModPackageSnapshot(mod dbtypes.Mod, metadata dbtypes.ModMetadata) dto.ModPackageSnapshot {
	return dto.ModPackageSnapshot{
		SourceType:         mappers.ToDTOModSourceType(mod.SourceType),
		OriginalSourcePath: mod.OriginalSourcePath,
		OriginalSourceName: mod.OriginalSourceName,
		FileCount:          mod.FileCount,
		DirectoryCount:     mod.DirectoryCount,
		TotalSizeBytes:     mod.TotalSizeBytes,
		DetectedMetadata: dto.ModDetectedMetadataSnapshot{
			Version:     metadata.DetectedVersion,
			Author:      metadata.DetectedAuthor,
			Description: metadata.DetectedDescription,
			SourceURL:   metadata.DetectedSourceURL,
		},
	}
}
