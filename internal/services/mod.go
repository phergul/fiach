package services

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/mod-manager/internal/fileignore"
	"github.com/phergul/mod-manager/internal/installconfig"
	"github.com/phergul/mod-manager/internal/modmetadata"
	"github.com/phergul/mod-manager/internal/services/dto"
	"github.com/phergul/mod-manager/internal/services/dto/mappers"
	"github.com/phergul/mod-manager/internal/storage"
)

type ModService struct {
	store            *storage.Store
	logger           *slog.Logger
	metadataRegistry *modmetadata.Registry
}

func NewModService(store *storage.Store, logger *slog.Logger) *ModService {
	return &ModService{
		store:            store,
		logger:           logger,
		metadataRegistry: modmetadata.DefaultRegistry(),
	}
}

func (s *ModService) ListMods(ctx context.Context, gameID int64) (mods []dto.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list mods: %w", err)
		}
	}()

	storedMods, err := s.store.ListMods(ctx, gameID)
	if err != nil {
		return nil, err
	}

	return mappers.ToDTOMods(storedMods), nil
}

func (s *ModService) RenameMod(ctx context.Context, modID int64, name string) (mod dto.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("rename mod: %w", err)
		}
	}()

	storedMod, err := s.store.RenameMod(ctx, modID, name)
	if err != nil {
		return dto.Mod{}, err
	}

	return mappers.ToDTOMod(storedMod), nil
}

func (s *ModService) GetModDeleteSummary(ctx context.Context, modID int64) (summary dto.ModDeleteSummary, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get mod delete summary: %w", err)
		}
	}()

	mod, found, err := s.store.GetMod(ctx, modID)
	if err != nil {
		return dto.ModDeleteSummary{}, err
	}
	if !found {
		return dto.ModDeleteSummary{}, fmt.Errorf("mod %d was not found", modID)
	}

	profileUsageCount, err := s.store.CountProfilesUsingMod(ctx, modID)
	if err != nil {
		return dto.ModDeleteSummary{}, err
	}

	isInAppliedProfile := false
	appliedState, appliedFound, err := s.store.GetAppliedProfileState(ctx, mod.GameID)
	if err != nil {
		return dto.ModDeleteSummary{}, err
	}
	if appliedFound {
		isInAppliedProfile, err = s.store.ProfileUsesMod(ctx, appliedState.ProfileID, modID)
		if err != nil {
			return dto.ModDeleteSummary{}, err
		}
	}

	return mappers.ToDTOModDeleteSummary(mod, profileUsageCount, isInAppliedProfile), nil
}

func (s *ModService) DeleteMod(ctx context.Context, modID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("delete mod: %w", err)
		}
	}()

	mod, found, err := s.store.GetMod(ctx, modID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("mod %d was not found", modID)
	}

	managedRoot, err := s.store.ResolveGameModStoragePath(ctx, mod.GameID, "")
	if err != nil {
		return err
	}
	if err := requireManagedModSourcePath(managedRoot, mod.SourcePath); err != nil {
		return err
	}

	if err := os.RemoveAll(mod.SourcePath); err != nil {
		return fmt.Errorf("remove managed mod files: %w", err)
	}

	if err := s.store.DeleteMod(ctx, modID); err != nil {
		return err
	}

	return nil
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

func requireManagedModSourcePath(managedRoot string, sourcePath string) error {
	managedRoot = strings.TrimSpace(managedRoot)
	sourcePath = strings.TrimSpace(sourcePath)
	if managedRoot == "" {
		return errors.New("managed mod storage path is required")
	}
	if sourcePath == "" {
		return errors.New("managed mod source path is required")
	}

	managedRootAbs, err := filepath.Abs(filepath.Clean(managedRoot))
	if err != nil {
		return fmt.Errorf("resolve managed mod storage path: %w", err)
	}
	sourcePathAbs, err := filepath.Abs(filepath.Clean(sourcePath))
	if err != nil {
		return fmt.Errorf("resolve managed mod source path: %w", err)
	}

	relativePath, err := filepath.Rel(managedRootAbs, sourcePathAbs)
	if err != nil {
		return fmt.Errorf("compare managed mod source path: %w", err)
	}
	if relativePath == "." || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("managed mod source path %q is outside managed storage %q", sourcePath, managedRoot)
	}

	return nil
}

func (s *ModService) ListImportStrategies(_ context.Context) (strategies []dto.StrategyDescriptor, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list import strategies: %w", err)
		}
	}()

	return mappers.ToDTOStrategyDescriptors(installconfig.SelectableStrategies()), nil
}

func managedModPathSize(path string) int64 {
	var total int64

	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}

	for _, entry := range entries {
		if fileignore.Has(entry.Name()) {
			continue
		}
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
