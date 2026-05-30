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
)

func (s *ModService) PreValidateImport(ctx context.Context, input dto.PreValidateImportInput) (err error) {
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
		return err
	}

	if err := source.Validate(); err != nil {
		return err
	}

	diag.complete("Mod import validation completed")

	return nil
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

	importResult, err := modimport.Import(
		ctx,
		s.store,
		input.GameID,
		input.Name,
		source,
		mappers.ToInstallStrategyType(input.StrategyType),
		input.TargetRelativePath,
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
		Mod:    mappers.ToDTOMod(importResult.Mod),
		Config: mappers.ToDTOModInstallConfig(importResult.Config),
	}, nil
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
