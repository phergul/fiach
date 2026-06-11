package modimport

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phergul/fiach/internal/installconfig"
	"github.com/phergul/fiach/internal/modmetadata"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

type UpdateStore interface {
	GetMod(ctx context.Context, modID int64) (dbtypes.Mod, bool, error)
	FindModByOriginalSourcePath(ctx context.Context, gameID int64, originalSourcePath string) (dbtypes.Mod, bool, error)
	GetGlobalModStorageRoot(ctx context.Context) (string, error)
	ResolveGameModStoragePath(ctx context.Context, gameID int64, globalRoot string) (string, error)
	GetModMetadata(ctx context.Context, modID int64) (dbtypes.ModMetadata, bool, error)
	GetModInstallConfig(ctx context.Context, modID int64) (dbtypes.ModInstallConfig, bool, error)
	UpdateModPackage(ctx context.Context, input dbtypes.UpdateModPackageInput) (dbtypes.Mod, error)
}

type UpdateResult struct {
	Before         dbtypes.Mod
	After          dbtypes.Mod
	BeforeMetadata dbtypes.ModMetadata
	AfterMetadata  dbtypes.ModMetadata
	MetadataError  error
	Warnings       []string
}

type preparedUpdate struct {
	before          dbtypes.Mod
	after           dbtypes.Mod
	beforeMetadata  dbtypes.ModMetadata
	afterMetadata   dbtypes.ModMetadata
	metadataError   error
	warnings        []string
	gameStoragePath string
	tempPath        string
	updateInput     dbtypes.UpdateModPackageInput
}

func PreviewUpdate(ctx context.Context, store UpdateStore, modID int64, source Source, options ImportOptions) (result UpdateResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("preview mod source update: %w", err)
		}
	}()

	prepared, cleanup, err := prepareUpdate(ctx, store, modID, source, options)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return UpdateResult{}, err
	}

	return prepared.result(), nil
}

func Update(ctx context.Context, store UpdateStore, modID int64, source Source, options ImportOptions) (result UpdateResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update mod source: %w", err)
		}
	}()

	prepared, cleanup, err := prepareUpdate(ctx, store, modID, source, options)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return UpdateResult{}, err
	}

	mod := prepared.before
	tempPath := prepared.tempPath
	backupPath, err := makeImportTempDir(prepared.gameStoragePath, filepath.Base(mod.SourcePath)+"-backup")
	if err != nil {
		return UpdateResult{}, err
	}
	if err := os.Remove(backupPath); err != nil {
		return UpdateResult{}, fmt.Errorf("prepare managed mod backup folder: %w", err)
	}
	removeBackup := true
	defer func() {
		if removeBackup {
			_ = os.RemoveAll(backupPath)
		}
	}()

	if err := os.Rename(mod.SourcePath, backupPath); err != nil {
		return UpdateResult{}, fmt.Errorf("move current managed mod folder to backup: %w", err)
	}
	currentMovedToBackup := true
	if err := os.Rename(tempPath, mod.SourcePath); err != nil {
		if currentMovedToBackup {
			_ = os.Rename(backupPath, mod.SourcePath)
			currentMovedToBackup = false
		}
		return UpdateResult{}, fmt.Errorf("move replacement managed mod folder into place: %w", err)
	}
	cleanup = nil
	replacementInPlace := true

	updatedMod, err := store.UpdateModPackage(ctx, prepared.updateInput)
	if err != nil {
		if replacementInPlace {
			_ = os.RemoveAll(mod.SourcePath)
			replacementInPlace = false
		}
		if currentMovedToBackup {
			_ = os.Rename(backupPath, mod.SourcePath)
			currentMovedToBackup = false
		}
		return UpdateResult{}, err
	}

	removeBackup = false
	if err := os.RemoveAll(backupPath); err != nil {
		return UpdateResult{}, fmt.Errorf("remove previous managed mod backup: %w", err)
	}

	return UpdateResult{
		Before:         prepared.before,
		After:          updatedMod,
		BeforeMetadata: prepared.beforeMetadata,
		AfterMetadata:  prepared.afterMetadata,
		MetadataError:  prepared.metadataError,
		Warnings:       prepared.warnings,
	}, nil
}

func prepareUpdate(ctx context.Context, store UpdateStore, modID int64, source Source, options ImportOptions) (prepared preparedUpdate, cleanup func(), err error) {
	if store == nil {
		return preparedUpdate{}, nil, errors.New("store is not configured")
	}
	if modID <= 0 {
		return preparedUpdate{}, nil, errors.New("mod ID must be positive")
	}
	if source == nil {
		return preparedUpdate{}, nil, errors.New("update source is required")
	}
	if err := source.Validate(); err != nil {
		return preparedUpdate{}, nil, err
	}

	mod, found, err := store.GetMod(ctx, modID)
	if err != nil {
		return preparedUpdate{}, nil, err
	}
	if !found {
		return preparedUpdate{}, nil, fmt.Errorf("mod %d was not found", modID)
	}

	if existing, found, err := store.FindModByOriginalSourcePath(ctx, mod.GameID, source.OriginalPath()); err != nil {
		return preparedUpdate{}, nil, err
	} else if found && existing.ID != mod.ID {
		return preparedUpdate{}, nil, fmt.Errorf("replacement source is already used by mod %d", existing.ID)
	}

	globalRoot, err := store.GetGlobalModStorageRoot(ctx)
	if err != nil {
		return preparedUpdate{}, nil, err
	}
	gameStoragePath, err := store.ResolveGameModStoragePath(ctx, mod.GameID, globalRoot)
	if err != nil {
		return preparedUpdate{}, nil, err
	}
	if err := requireManagedSourcePath(gameStoragePath, mod.SourcePath); err != nil {
		return preparedUpdate{}, nil, err
	}
	if pathContains(source.OriginalPath(), gameStoragePath) {
		return preparedUpdate{}, nil, fmt.Errorf("source %q is inside the managed mod storage folder %q", source.OriginalPath(), gameStoragePath)
	}
	if pathContains(gameStoragePath, source.OriginalPath()) {
		return preparedUpdate{}, nil, fmt.Errorf("source %q contains the managed mod storage folder %q", source.OriginalPath(), gameStoragePath)
	}

	beforeMetadata, metadataFound, err := store.GetModMetadata(ctx, mod.ID)
	if err != nil {
		return preparedUpdate{}, nil, err
	}
	if !metadataFound {
		beforeMetadata = dbtypes.ModMetadata{ModID: mod.ID}
	}

	tempPath, err := makeImportTempDir(gameStoragePath, filepath.Base(mod.SourcePath))
	if err != nil {
		return preparedUpdate{}, nil, err
	}
	cleanup = func() {
		if tempPath != "" {
			_ = os.RemoveAll(tempPath)
		}
	}

	if err := source.Materialize(tempPath); err != nil {
		cleanup()
		return preparedUpdate{}, nil, err
	}

	config, configFound, err := store.GetModInstallConfig(ctx, mod.ID)
	if err != nil {
		return preparedUpdate{}, cleanup, err
	}
	if !configFound {
		return preparedUpdate{}, cleanup, fmt.Errorf("mod %d install configuration was not found", mod.ID)
	}
	strategyWarnings, err := validateMaterializedStrategy(installconfig.StrategyType(config.StrategyType), tempPath)
	if err != nil {
		return preparedUpdate{}, cleanup, err
	}

	metadata, metadataErr := parseImportMetadata(ctx, options.MetadataRegistry, mod.GameID, source.Type(), tempPath)
	updateInput := updateInputFromMetadata(mod, source, metadata)
	afterMetadataInput := dbtypes.ModMetadataDetectedInput{
		Version:     metadata.Version,
		Author:      metadata.Author,
		Description: metadata.Description,
		SourceURL:   metadata.SourceURL,
	}
	if metadataErr != nil {
		updateInput = updateInputFromExisting(mod, source, beforeMetadata)
		afterMetadataInput = dbtypes.ModMetadataDetectedInput{
			Version:     beforeMetadata.DetectedVersion,
			Author:      beforeMetadata.DetectedAuthor,
			Description: beforeMetadata.DetectedDescription,
			SourceURL:   beforeMetadata.DetectedSourceURL,
		}
	}

	afterMetadata := beforeMetadata
	afterMetadata.ModID = mod.ID
	afterMetadata.DetectedVersion = afterMetadataInput.Version
	afterMetadata.DetectedAuthor = afterMetadataInput.Author
	afterMetadata.DetectedDescription = afterMetadataInput.Description
	afterMetadata.DetectedSourceURL = afterMetadataInput.SourceURL

	after := mod
	after.SourceType = updateInput.SourceType
	after.OriginalSourcePath = updateInput.OriginalSourcePath
	after.OriginalSourceName = updateInput.OriginalSourceName
	after.FileCount = updateInput.FileCount
	after.DirectoryCount = updateInput.DirectoryCount
	after.TotalSizeBytes = updateInput.TotalSizeBytes
	after.MetadataJSON = updateInput.MetadataJSON

	return preparedUpdate{
		before:          mod,
		after:           after,
		beforeMetadata:  beforeMetadata,
		afterMetadata:   afterMetadata,
		metadataError:   metadataErr,
		warnings:        strategyWarnings,
		gameStoragePath: gameStoragePath,
		tempPath:        tempPath,
		updateInput:     updateInput,
	}, cleanup, nil
}

func (p preparedUpdate) result() UpdateResult {
	return UpdateResult{
		Before:         p.before,
		After:          p.after,
		BeforeMetadata: p.beforeMetadata,
		AfterMetadata:  p.afterMetadata,
		MetadataError:  p.metadataError,
		Warnings:       p.warnings,
	}
}

func updateInputFromMetadata(mod dbtypes.Mod, source Source, metadata modmetadata.Metadata) dbtypes.UpdateModPackageInput {
	return dbtypes.UpdateModPackageInput{
		ModID:              mod.ID,
		SourceType:         source.Type(),
		OriginalSourcePath: source.OriginalPath(),
		OriginalSourceName: source.OriginalName(),
		FileCount:          metadata.FileCount,
		DirectoryCount:     metadata.DirectoryCount,
		TotalSizeBytes:     metadata.TotalSizeBytes,
		MetadataJSON:       metadata.JSON,
		DetectedMetadata: dbtypes.ModMetadataDetectedInput{
			Version:     metadata.Version,
			Author:      metadata.Author,
			Description: metadata.Description,
			SourceURL:   metadata.SourceURL,
		},
	}
}

func updateInputFromExisting(mod dbtypes.Mod, source Source, metadata dbtypes.ModMetadata) dbtypes.UpdateModPackageInput {
	return dbtypes.UpdateModPackageInput{
		ModID:              mod.ID,
		SourceType:         source.Type(),
		OriginalSourcePath: source.OriginalPath(),
		OriginalSourceName: source.OriginalName(),
		FileCount:          mod.FileCount,
		DirectoryCount:     mod.DirectoryCount,
		TotalSizeBytes:     mod.TotalSizeBytes,
		MetadataJSON:       mod.MetadataJSON,
		DetectedMetadata: dbtypes.ModMetadataDetectedInput{
			Version:     metadata.DetectedVersion,
			Author:      metadata.DetectedAuthor,
			Description: metadata.DetectedDescription,
			SourceURL:   metadata.DetectedSourceURL,
		},
	}
}

func requireManagedSourcePath(managedRoot string, sourcePath string) error {
	if managedRoot == "" {
		return errors.New("managed mod storage path is required")
	}
	if sourcePath == "" {
		return errors.New("managed mod source path is required")
	}
	if !pathContains(sourcePath, managedRoot) {
		return fmt.Errorf("managed mod source path %q is outside managed storage %q", sourcePath, managedRoot)
	}

	return nil
}
