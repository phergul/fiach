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

func (s *ModService) ResolveImportSourceDuplicates(ctx context.Context, input dto.ResolveImportSourceDuplicatesInput) (result dto.ResolveImportSourceDuplicatesResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("resolve import source duplicates: %w", err)
		}
	}()

	if input.GameID <= 0 {
		return dto.ResolveImportSourceDuplicatesResult{}, errors.New("game ID must be positive")
	}

	result.Items = make([]dto.ImportSourceDuplicateStatus, 0, len(input.Sources))
	canonicalPaths := make([]string, 0, len(input.Sources))

	for _, sourceRef := range input.Sources {
		status := dto.ImportSourceDuplicateStatus{
			SourceType: sourceRef.SourceType,
			SourcePath: sourceRef.SourcePath,
		}

		source, sourceErr := importSource(sourceRef.SourceType, sourceRef.SourcePath)
		if sourceErr != nil {
			message := modImportUserError(sourceErr).Error()
			status.Error = &message
			result.Items = append(result.Items, status)
			continue
		}

		canonicalPath := source.OriginalPath()
		status.CanonicalPath = canonicalPath
		canonicalPaths = append(canonicalPaths, canonicalPath)
		result.Items = append(result.Items, status)
	}

	if len(canonicalPaths) == 0 {
		return result, nil
	}

	modsByPath, err := s.store.FindModsByOriginalSourcePaths(ctx, input.GameID, canonicalPaths)
	if err != nil {
		return dto.ResolveImportSourceDuplicatesResult{}, err
	}

	for index := range result.Items {
		status := &result.Items[index]
		if status.Error != nil || status.CanonicalPath == "" {
			continue
		}

		existingMod, found := modsByPath[status.CanonicalPath]
		if !found {
			continue
		}

		status.IsDuplicate = true
		status.ExistingModID = &existingMod.ID
		status.ExistingModName = &existingMod.Name
	}

	return result, nil
}

func (s *ModService) PreValidateImport(ctx context.Context, input dto.PreValidateImportInput) (result dto.PreValidateImportResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationPreValidateMod, "Mod import validation started",
		slog.String("source_type", string(input.SourceType)),
		diagnostics.PathAttr("source_path", input.SourcePath),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Mod import validation failed", err, modImportUserError)
		}
	}()

	source, err := importSource(input.SourceType, input.SourcePath)
	if err != nil {
		return dto.PreValidateImportResult{}, err
	}

	if err := source.Validate(ctx); err != nil {
		return dto.PreValidateImportResult{}, err
	}

	tempPath, err := os.MkdirTemp("", "fiach-import-inspection-*")
	if err != nil {
		return dto.PreValidateImportResult{}, fmt.Errorf("create import inspection folder: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempPath)
	}()

	if err := source.Materialize(ctx, tempPath); err != nil {
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
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationDetectImportTargets, "Import target detection started",
		slog.Int64("game_id", gameID),
		slog.String("strategy_type", string(strategyType)),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Import target detection failed", err, modImportUserError)
		}
	}()

	if gameID <= 0 {
		return dto.ImportTargetDetectionResult{}, errors.New("game ID must be positive")
	}
	if strategyType != dto.StrategyTypeUnrealPak {
		diag.complete("Import target detection completed",
			slog.Int("candidate_count", 0),
			slog.Int("warning_count", 0),
		)
		return dto.ImportTargetDetectionResult{
			Candidates: []string{},
			Warnings:   []string{},
		}, nil
	}

	game, err := s.store.GetStoredGame(ctx, gameID)
	if err != nil {
		return dto.ImportTargetDetectionResult{}, err
	}
	diag.attrs = append(diag.attrs, slog.String("game_name", game.Name))
	detection, err := unrealpak.DetectTargets(game.InstallPath)
	if err != nil {
		return dto.ImportTargetDetectionResult{}, err
	}

	result = dto.ImportTargetDetectionResult{
		Candidates: detection.Candidates,
		Warnings:   detection.Warnings,
	}
	diag.complete("Import target detection completed",
		slog.Int("candidate_count", len(result.Candidates)),
		slog.Int("warning_count", len(result.Warnings)),
	)

	return result, nil
}

func (s *ModService) PreviewImportConfiguration(ctx context.Context, input dto.PreviewImportConfigurationInput) (preview dto.Preview, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationPreviewMod, "Mod import preview started",
		slog.String("source_type", string(input.SourceType)),
		slog.String("strategy_type", string(input.StrategyType)),
		diagnostics.PathAttr("source_path", input.SourcePath),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Mod import preview failed", err, modImportUserError)
		}
	}()

	source, err := importSource(input.SourceType, input.SourcePath)
	if err != nil {
		return dto.Preview{}, err
	}
	if err := source.Validate(ctx); err != nil {
		return dto.Preview{}, err
	}

	tempPath, err := os.MkdirTemp("", "fiach-import-preview-*")
	if err != nil {
		return dto.Preview{}, fmt.Errorf("create import preview folder: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempPath)
	}()

	if err := source.Materialize(ctx, tempPath); err != nil {
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
			err = diag.failWithMappedError("Mod import failed", err, modImportUserError)
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
			TagIDs:           input.TagIDs,
			NewTags:          mappers.ToStorageCreateTagInputs(input.NewTags),
		},
	)
	if err != nil {
		return dto.ImportModResult{}, err
	}
	if importResult.MetadataError != nil {
		diag.warnEvent("metadata_unavailable", "Mod metadata unavailable",
			slog.Int64("game_id", input.GameID),
			slog.Int64("mod_id", importResult.Mod.ID),
			slog.String("mod_name", importResult.Mod.Name),
			slog.String("source_type", string(input.SourceType)),
			diagnostics.PathAttr("source_path", input.SourcePath),
			diagnostics.ErrorAttr(importResult.MetadataError),
		)
	}

	diag.complete("Mod import completed",
		slog.Int64("mod_id", importResult.Mod.ID),
		slog.String("mod_name", importResult.Mod.Name),
	)

	resultMod := mappers.ToDTOMod(importResult.Mod)
	resultMod.Tags = mappers.ToDTOTags(importResult.Tags)
	return dto.ImportModResult{
		Mod:      resultMod,
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
			err = diag.failWithMappedError("Mod update preview failed", err, modImportUserError)
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
		slog.String("mod_name", updateResult.After.Name),
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
			err = diag.failWithMappedError("Mod update failed", err, modImportUserError)
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
		diag.warnEvent("metadata_unavailable", "Mod metadata unavailable",
			slog.Int64("game_id", updateResult.After.GameID),
			slog.Int64("mod_id", updateResult.After.ID),
			slog.String("mod_name", updateResult.After.Name),
			slog.String("source_type", string(input.SourceType)),
			diagnostics.PathAttr("source_path", input.SourcePath),
			diagnostics.ErrorAttr(updateResult.MetadataError),
		)
	}

	diag.complete("Mod update completed",
		slog.Int64("game_id", updateResult.After.GameID),
		slog.Int64("mod_id", updateResult.After.ID),
		slog.String("mod_name", updateResult.After.Name),
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

	resultMod := mappers.ToDTOMod(updateResult.After)
	tagsByModID, err := s.store.ListTagsForMods(ctx, []int64{updateResult.After.ID})
	if err != nil {
		return dto.UpdateModResult{}, err
	}
	resultMod.Tags = mappers.ToDTOTags(tagsByModID[updateResult.After.ID])

	return dto.UpdateModResult{
		Mod:                resultMod,
		Before:             mappers.ToModPackageSnapshot(updateResult.Before, updateResult.BeforeMetadata),
		After:              mappers.ToModPackageSnapshot(updateResult.After, updateResult.AfterMetadata),
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
