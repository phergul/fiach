package modimport

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phergul/fiach/internal/modmetadata"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

type UpdateStore interface {
	GetMod(ctx context.Context, modID int64) (dbtypes.Mod, bool, error)
	FindModByOriginalSourcePath(ctx context.Context, gameID int64, originalSourcePath string) (dbtypes.Mod, bool, error)
	GetGlobalModStorageRoot(ctx context.Context) (string, error)
	ResolveGameModStoragePath(ctx context.Context, gameID int64, globalRoot string) (string, error)
	GetModMetadata(ctx context.Context, modID int64) (dbtypes.ModMetadata, bool, error)
	UpdateModPackage(ctx context.Context, input dbtypes.UpdateModPackageInput) (dbtypes.Mod, error)
}

type UpdateResult struct {
	Before         dbtypes.Mod
	After          dbtypes.Mod
	BeforeMetadata dbtypes.ModMetadata
	AfterMetadata  dbtypes.ModMetadata
	MetadataError  error
}

func Update(ctx context.Context, store UpdateStore, modID int64, source Source, options ImportOptions) (result UpdateResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update mod source: %w", err)
		}
	}()

	if store == nil {
		return UpdateResult{}, errors.New("store is not configured")
	}
	if modID <= 0 {
		return UpdateResult{}, errors.New("mod ID must be positive")
	}
	if source == nil {
		return UpdateResult{}, errors.New("update source is required")
	}
	if err := source.Validate(); err != nil {
		return UpdateResult{}, err
	}

	mod, found, err := store.GetMod(ctx, modID)
	if err != nil {
		return UpdateResult{}, err
	}
	if !found {
		return UpdateResult{}, fmt.Errorf("mod %d was not found", modID)
	}

	if existing, found, err := store.FindModByOriginalSourcePath(ctx, mod.GameID, source.OriginalPath()); err != nil {
		return UpdateResult{}, err
	} else if found && existing.ID != mod.ID {
		return UpdateResult{}, fmt.Errorf("replacement source is already used by mod %d", existing.ID)
	}

	globalRoot, err := store.GetGlobalModStorageRoot(ctx)
	if err != nil {
		return UpdateResult{}, err
	}
	gameStoragePath, err := store.ResolveGameModStoragePath(ctx, mod.GameID, globalRoot)
	if err != nil {
		return UpdateResult{}, err
	}
	if err := requireManagedSourcePath(gameStoragePath, mod.SourcePath); err != nil {
		return UpdateResult{}, err
	}
	if pathContains(source.OriginalPath(), gameStoragePath) {
		return UpdateResult{}, fmt.Errorf("source %q is inside the managed mod storage folder %q", source.OriginalPath(), gameStoragePath)
	}
	if pathContains(gameStoragePath, source.OriginalPath()) {
		return UpdateResult{}, fmt.Errorf("source %q contains the managed mod storage folder %q", source.OriginalPath(), gameStoragePath)
	}

	beforeMetadata, metadataFound, err := store.GetModMetadata(ctx, mod.ID)
	if err != nil {
		return UpdateResult{}, err
	}
	if !metadataFound {
		beforeMetadata = dbtypes.ModMetadata{ModID: mod.ID}
	}

	tempPath, err := makeImportTempDir(gameStoragePath, filepath.Base(mod.SourcePath))
	if err != nil {
		return UpdateResult{}, err
	}
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.RemoveAll(tempPath)
		}
	}()

	if err := source.Materialize(tempPath); err != nil {
		return UpdateResult{}, err
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

	backupPath, err := makeImportTempDir(gameStoragePath, filepath.Base(mod.SourcePath)+"-backup")
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
	removeTemp = false
	replacementInPlace := true

	updatedMod, err := store.UpdateModPackage(ctx, updateInput)
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

	afterMetadata := beforeMetadata
	afterMetadata.ModID = mod.ID
	afterMetadata.DetectedVersion = afterMetadataInput.Version
	afterMetadata.DetectedAuthor = afterMetadataInput.Author
	afterMetadata.DetectedDescription = afterMetadataInput.Description
	afterMetadata.DetectedSourceURL = afterMetadataInput.SourceURL

	return UpdateResult{
		Before:         mod,
		After:          updatedMod,
		BeforeMetadata: beforeMetadata,
		AfterMetadata:  afterMetadata,
		MetadataError:  metadataErr,
	}, nil
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
