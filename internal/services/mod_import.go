package services

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/phergul/mod-manager/internal/installconfig"
	"github.com/phergul/mod-manager/internal/modimport"
	"github.com/phergul/mod-manager/internal/storage"
)

type PreviewImportConfigurationInput struct {
	GameID             int64
	SourceType         storage.ModSourceType
	SourcePath         string
	StrategyType       installconfig.StrategyType
	TargetRelativePath string
}

type ImportConfiguredModInput struct {
	GameID             int64
	Name               string
	SourceType         storage.ModSourceType
	SourcePath         string
	StrategyType       installconfig.StrategyType
	TargetRelativePath string
}

type ImportConfiguredModResult struct {
	Mod    storage.Mod
	Config storage.ModInstallConfig
}

func (s *ModService) ImportModFolder(gameID int64, name string, sourceFolderPath string) (mod storage.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("import mod folder: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return storage.Mod{}, errors.New("storage is not configured")
	}

	source, err := modimport.NewFolderSource(sourceFolderPath)
	if err != nil {
		return storage.Mod{}, err
	}

	return modimport.Import(context.Background(), s.store, gameID, name, source)
}

func (s *ModService) PreviewImportConfiguration(input PreviewImportConfigurationInput) (preview installconfig.Preview, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("preview import configuration: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return installconfig.Preview{}, errors.New("storage is not configured")
	}

	source, err := importSource(input.SourceType, input.SourcePath)
	if err != nil {
		return installconfig.Preview{}, err
	}
	if err := source.Validate(); err != nil {
		return installconfig.Preview{}, err
	}

	tempPath, err := os.MkdirTemp("", "mod-manager-import-preview-*")
	if err != nil {
		return installconfig.Preview{}, fmt.Errorf("create import preview folder: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempPath)
	}()

	if err := source.Materialize(tempPath); err != nil {
		return installconfig.Preview{}, err
	}

	return installconfig.BuildPreview(installconfig.PreviewInput{
		SourcePath:         tempPath,
		StrategyType:       input.StrategyType,
		TargetRelativePath: input.TargetRelativePath,
		FileCap:            installconfig.DefaultPreviewFileCap,
	})
}

func (s *ModService) ImportConfiguredMod(input ImportConfiguredModInput) (result ImportConfiguredModResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("import configured mod: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return ImportConfiguredModResult{}, errors.New("storage is not configured")
	}

	source, err := importSource(input.SourceType, input.SourcePath)
	if err != nil {
		return ImportConfiguredModResult{}, err
	}

	importResult, err := modimport.ImportConfigured(
		context.Background(),
		s.store,
		input.GameID,
		input.Name,
		source,
		input.StrategyType,
		input.TargetRelativePath,
	)
	if err != nil {
		return ImportConfiguredModResult{}, err
	}

	return ImportConfiguredModResult{
		Mod:    importResult.Mod,
		Config: importResult.Config,
	}, nil
}

func (s *ModService) ImportModArchive(gameID int64, name string, archiveFilePath string) (mod storage.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("import mod archive: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return storage.Mod{}, errors.New("storage is not configured")
	}

	source, err := modimport.NewArchiveSource(archiveFilePath)
	if err != nil {
		return storage.Mod{}, err
	}

	return modimport.Import(context.Background(), s.store, gameID, name, source)
}

func importSource(sourceType storage.ModSourceType, sourcePath string) (modimport.Source, error) {
	switch sourceType {
	case storage.ModSourceTypeFolder:
		return modimport.NewFolderSource(sourcePath)
	case storage.ModSourceTypeArchive:
		return modimport.NewArchiveSource(sourcePath)
	case "":
		return nil, errors.New("import source type is required")
	default:
		return nil, fmt.Errorf("unsupported import source type %q", sourceType)
	}
}
