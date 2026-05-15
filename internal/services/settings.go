package services

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/phergul/mod-manager/internal/storage"
)

type SettingsService struct {
	store *storage.Store
}

func NewSettingsService(store *storage.Store) *SettingsService {
	return &SettingsService{
		store: store,
	}
}

func (s *SettingsService) GetGlobalModStorageRoot() (root string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get global mod storage root: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return "", errors.New("storage is not configured")
	}

	return s.store.GetGlobalModStorageRoot(context.Background())
}

func (s *SettingsService) SetGlobalModStorageRoot(path string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("set global mod storage root: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return errors.New("storage is not configured")
	}

	return s.store.SetGlobalModStorageRoot(context.Background(), path)
}

func (s *SettingsService) SetGameModStoragePathOverride(gameID int64, path string) (game storage.StoredGame, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("set game mod storage path override: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return storage.StoredGame{}, errors.New("storage is not configured")
	}

	return s.store.SetGameModStoragePathOverride(context.Background(), gameID, path)
}

func (s *SettingsService) ResolveGameModStoragePath(gameID int64) (path string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("resolve game mod storage path: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return "", errors.New("storage is not configured")
	}

	globalRoot, err := s.store.GetGlobalModStorageRoot(context.Background())
	if err != nil {
		return "", err
	}

	return s.store.ResolveGameModStoragePath(context.Background(), gameID, globalRoot)
}

func (s *SettingsService) EnsureGameModStoragePath(gameID int64) (path string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("ensure game mod storage path: %w", err)
		}
	}()

	path, err = s.ResolveGameModStoragePath(gameID)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("create game mod storage folder: %w", err)
	}

	return path, nil
}
