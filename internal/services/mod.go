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

	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/fileignore"
	"github.com/phergul/fiach/internal/installconfig"
	"github.com/phergul/fiach/internal/modmetadata"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
	"github.com/phergul/fiach/internal/storage"
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

	mods = make([]dto.Mod, 0, len(storedMods))
	for _, storedMod := range storedMods {
		metadata, found, err := s.store.GetModMetadata(ctx, storedMod.ID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("mod %d metadata was not found", storedMod.ID)
		}
		mods = append(mods, mappers.ToDTOModWithMetadata(storedMod, metadata))
	}

	return mods, nil
}

func (s *ModService) RenameMod(ctx context.Context, modID int64, name string) (mod dto.Mod, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationRenameMod, "Mod rename started",
		slog.Int64("mod_id", modID),
	)
	defer func() {
		if err != nil {
			diag.fail("Mod rename failed", err)
			err = fmt.Errorf("rename mod: %w", err)
		}
	}()

	storedMod, err := s.store.RenameMod(ctx, modID, name)
	if err != nil {
		return dto.Mod{}, err
	}

	mod = mappers.ToDTOMod(storedMod)
	diag.complete("Mod rename completed",
		slog.Int64("game_id", storedMod.GameID),
		slog.String("mod_name", storedMod.Name),
	)

	return mod, nil
}

func (s *ModService) GetModDeleteSummary(ctx context.Context, modID int64) (summary dto.ModDeleteSummary, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationGetModDeleteSummary, "Mod delete summary started",
		slog.Int64("mod_id", modID),
	)
	defer func() {
		if err != nil {
			diag.fail("Mod delete summary failed", err)
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

	summary = mappers.ToDTOModDeleteSummary(mod, profileUsageCount, isInAppliedProfile)
	diag.complete("Mod delete summary completed",
		slog.Int64("game_id", mod.GameID),
		slog.String("mod_name", mod.Name),
		slog.Int64("profile_usage_count", profileUsageCount),
		slog.Bool("is_in_applied_profile", isInAppliedProfile),
	)

	return summary, nil
}

func (s *ModService) DeleteMod(ctx context.Context, modID int64) (err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationDeleteMod, "Mod delete started",
		slog.Int64("mod_id", modID),
	)
	defer func() {
		if err != nil {
			diag.fail("Mod delete failed", err)
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
	diag.attrs = append(diag.attrs,
		slog.Int64("game_id", mod.GameID),
		slog.String("mod_name", mod.Name),
		diagnostics.PathAttr("source_path", mod.SourcePath),
	)

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

	diag.complete("Mod delete completed")

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
