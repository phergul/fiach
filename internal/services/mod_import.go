package services

import (
	"context"
	"errors"
	"fmt"
	"os"

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
	defer func() {
		if err != nil {
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
		toInstallStrategyType(input.StrategyType),
		input.TargetRelativePath,
	)
	if err != nil {
		return dto.ImportModResult{}, err
	}

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
