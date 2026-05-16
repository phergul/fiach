package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/phergul/mod-manager/internal/modimport"
	"github.com/phergul/mod-manager/internal/storage"
)

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
