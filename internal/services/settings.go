package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
	"github.com/phergul/fiach/internal/storage"
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
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationSetGlobalModStorageRoot, "Global mod storage root update started",
		diagnostics.PathAttr("storage_root", path),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Global mod storage root update failed", err, settingsUserError)
		}
	}()

	if err := s.store.SetGlobalModStorageRoot(ctx, path); err != nil {
		return err
	}

	diag.complete("Global mod storage root update completed")

	return nil
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
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationSetTheme, "Theme update started")
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Theme update failed", err, settingsUserError)
		}
	}()

	themeID = strings.TrimSpace(themeID)
	if themeID == "" {
		return errors.New("theme ID is required")
	}

	if err := s.store.SetThemeID(ctx, themeID); err != nil {
		return err
	}

	diag.complete("Theme update completed",
		slog.String("theme_id", themeID),
	)

	return nil
}

func (s *SettingsService) SetGameModStoragePathOverride(ctx context.Context, gameID int64, path string) (game dto.StoredGame, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationSetGameModStorageOverride, "Game mod storage override update started",
		slog.Int64("game_id", gameID),
		diagnostics.PathAttr("storage_override", path),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Game mod storage override update failed", err, settingsUserError)
		}
	}()

	storedGame, err := s.store.SetGameModStoragePathOverride(ctx, gameID, path)
	if err != nil {
		return dto.StoredGame{}, err
	}

	game = mappers.ToDTOStoredGame(storedGame)
	diag.complete("Game mod storage override update completed",
		slog.String("game_name", storedGame.Name),
	)

	return game, nil
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
