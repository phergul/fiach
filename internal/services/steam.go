package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/phergul/mod-manager/internal/steam"
	"github.com/phergul/mod-manager/internal/storage"
)

const SteamInstallPathSettingKey = "steam.install_path"

type SteamService struct {
	store *storage.Store
}

func NewSteamService(store *storage.Store) *SteamService {
	return &SteamService{
		store: store,
	}
}

func (s *SteamService) LocateSteamInstallation() (*steam.SteamPaths, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("locate Steam installation: storage is not configured")
	}

	manualPath, found, err := s.store.GetSetting(context.Background(), SteamInstallPathSettingKey)
	if err != nil {
		return nil, fmt.Errorf("locate Steam installation: %w", err)
	}
	if !found {
		manualPath = ""
	}

	paths, err := steam.FindSteamPaths(manualPath)
	if err != nil {
		if errors.Is(err, steam.ErrSteamNotFound) {
			return nil, fmt.Errorf("Steam installation could not be found. Configure a valid Steam path in settings or install Steam in a standard location: %w", err)
		}

		return nil, fmt.Errorf("locate Steam installation: %w", err)
	}

	return paths, nil
}
