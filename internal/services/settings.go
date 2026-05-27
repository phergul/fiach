package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/phergul/mod-manager/internal/services/dto"
	"github.com/phergul/mod-manager/internal/services/dto/mappers"
	"github.com/phergul/mod-manager/internal/storage"
)

type SettingsService struct {
	store  *storage.Store
	logger *slog.Logger
}

func NewSettingsService(store *storage.Store, logger *slog.Logger) *SettingsService {
	return &SettingsService{
		store:  store,
		logger: logger,
	}
}

func (s *SettingsService) GetGlobalModStorageRoot(ctx context.Context) (root string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get global mod storage root: %w", err)
		}
	}()

	return s.store.GetGlobalModStorageRoot(ctx)
}

func (s *SettingsService) SetGlobalModStorageRoot(ctx context.Context, path string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("set global mod storage root: %w", err)
		}
	}()

	return s.store.SetGlobalModStorageRoot(ctx, path)
}

func (s *SettingsService) GetThemeID(ctx context.Context) (themeID string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get theme ID: %w", err)
		}
	}()

	return s.store.GetThemeID(ctx)
}

func (s *SettingsService) SetThemeID(ctx context.Context, themeID string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("set theme ID: %w", err)
		}
	}()

	themeID = strings.TrimSpace(themeID)
	if themeID == "" {
		return errors.New("theme ID is required")
	}

	return s.store.SetThemeID(ctx, themeID)
}

func (s *SettingsService) SetGameModStoragePathOverride(ctx context.Context, gameID int64, path string) (game dto.StoredGame, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("set game mod storage path override: %w", err)
		}
	}()

	storedGame, err := s.store.SetGameModStoragePathOverride(ctx, gameID, path)
	if err != nil {
		return dto.StoredGame{}, err
	}

	return mappers.ToDTOStoredGame(storedGame), nil
}

func (s *SettingsService) ResolveGameModStoragePath(ctx context.Context, gameID int64) (path string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("resolve game mod storage path: %w", err)
		}
	}()

	globalRoot, err := s.store.GetGlobalModStorageRoot(ctx)
	if err != nil {
		return "", err
	}

	return s.store.ResolveGameModStoragePath(ctx, gameID, globalRoot)
}

func (s *SettingsService) EnsureGameModStoragePath(ctx context.Context, gameID int64) (path string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("ensure game mod storage path: %w", err)
		}
	}()

	path, err = s.ResolveGameModStoragePath(ctx, gameID)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("create game mod storage folder: %w", err)
	}

	return path, nil
}
