package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/phergul/mod-manager/internal/diagnostics"
	"github.com/phergul/mod-manager/internal/installconfig"
	"github.com/phergul/mod-manager/internal/modimport"
	"github.com/phergul/mod-manager/internal/services/dto"
	"github.com/phergul/mod-manager/internal/storage/dbtypes"
)

func (s *ModService) PreviewImportConfiguration(_ context.Context, input dto.PreviewImportConfigurationInput) (preview dto.Preview, err error) {
	defer func() {
		if err != nil {
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

	tempPath, err := os.MkdirTemp("", "mod-manager-import-preview-*")
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
		StrategyType:       toInstallStrategyType(input.StrategyType),
		TargetRelativePath: input.TargetRelativePath,
		FileCap:            installconfig.DefaultPreviewFileCap,
	})
	if err != nil {
		return dto.Preview{}, err
	}

	return toDTOPreview(previewResult), nil
}

func (s *ModService) ImportMod(ctx context.Context, input dto.ImportModInput) (result dto.ImportModResult, err error) {
	startedAt := time.Now()
	defer func() {
		if err != nil {
			s.logger.ErrorContext(ctx, "Mod import failed",
				slog.String("operation", diagnostics.OperationImportMod),
				slog.String("event", diagnostics.EventFailed),
				slog.Int64("game_id", input.GameID),
				slog.String("source_type", string(input.SourceType)),
				slog.String("strategy_type", string(input.StrategyType)),
				diagnostics.PathAttr("source_path", input.SourcePath),
				diagnostics.DurationAttr(startedAt),
				diagnostics.ErrorAttr(err),
			)
			err = fmt.Errorf("import mod: %w", err)
		}
	}()

	s.logger.InfoContext(ctx, "Mod import started",
		slog.String("operation", diagnostics.OperationImportMod),
		slog.String("event", diagnostics.EventStarted),
		slog.Int64("game_id", input.GameID),
		slog.String("source_type", string(input.SourceType)),
		slog.String("strategy_type", string(input.StrategyType)),
		diagnostics.PathAttr("source_path", input.SourcePath),
	)

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
		toInstallStrategyType(input.StrategyType),
		input.TargetRelativePath,
	)
	if err != nil {
		return dto.ImportModResult{}, err
	}

	s.logger.InfoContext(ctx, "Mod import completed",
		slog.String("operation", diagnostics.OperationImportMod),
		slog.String("event", diagnostics.EventCompleted),
		slog.Int64("game_id", input.GameID),
		slog.Int64("mod_id", importResult.Mod.ID),
		slog.String("source_type", string(input.SourceType)),
		slog.String("strategy_type", string(input.StrategyType)),
		diagnostics.PathAttr("source_path", input.SourcePath),
		diagnostics.DurationAttr(startedAt),
	)

	return dto.ImportModResult{
		Mod:    toDTOMod(importResult.Mod),
		Config: toDTOModInstallConfig(importResult.Config),
	}, nil
}

func importSource(sourceType dto.ModSourceType, sourcePath string) (modimport.Source, error) {
	switch toDBModSourceType(sourceType) {
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
