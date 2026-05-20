package services

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/phergul/mod-manager/internal/installconfig"
	"github.com/phergul/mod-manager/internal/storage"
)

type ModService struct {
	store *storage.Store
}

func NewModService(store *storage.Store) *ModService {
	return &ModService{
		store: store,
	}
}

func (s *ModService) ListMods(gameID int64) (mods []storage.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list mods: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return nil, errors.New("storage is not configured")
	}

	return s.store.ListMods(context.Background(), gameID)
}

func (s *ModService) GetGameManagedModStorageUsage(gameID int64) (bytes int64, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get game managed mod storage usage: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return 0, errors.New("storage is not configured")
	}

	mods, err := s.store.ListMods(context.Background(), gameID)
	if err != nil {
		return 0, err
	}

	for _, mod := range mods {
		bytes += managedModPathSize(mod.SourcePath)
	}

	return bytes, nil
}

func (s *ModService) ListImportStrategies() (strategies []installconfig.StrategyDescriptor, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list import strategies: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return nil, errors.New("storage is not configured")
	}

	return installconfig.SelectableStrategies(), nil
}

func managedModPathSize(path string) int64 {
	var total int64

	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}

	for _, entry := range entries {
		if entry.Type()&fs.ModeSymlink != 0 {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.IsDir() {
			total += managedModPathSize(filepath.Join(path, entry.Name()))
			continue
		}

		if info.Mode().IsRegular() {
			total += info.Size()
		}
	}

	return total
}
