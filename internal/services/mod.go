package services

import (
	"context"
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

func (s *ModService) ListMods(ctx context.Context, gameID int64) (mods []storage.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list mods: %w", err)
		}
	}()

	return s.store.ListMods(ctx, gameID)
}

func (s *ModService) GetGameManagedModStorageUsage(ctx context.Context, gameID int64) (bytes int64, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get game managed mod storage usage: %w", err)
		}
	}()

	mods, err := s.store.ListMods(ctx, gameID)
	if err != nil {
		return 0, err
	}

	for _, mod := range mods {
		bytes += managedModPathSize(mod.SourcePath)
	}

	return bytes, nil
}

func (s *ModService) ListImportStrategies(_ context.Context) (strategies []installconfig.StrategyDescriptor, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list import strategies: %w", err)
		}
	}()

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
