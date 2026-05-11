package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

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
	paths, err := s.locateSteamInstallation()
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (s *SteamService) GetSteamLibraries() ([]string, error) {
	paths, err := s.locateSteamInstallation()
	if err != nil {
		return nil, err
	}

	libraries, err := steam.ParseLibraryFolders(paths)
	if err != nil {
		return nil, fmt.Errorf("get Steam libraries: %w", err)
	}

	return libraries, nil
}

func (s *SteamService) GetInstalledSteamGames() ([]steam.Game, error) {
	libraries, err := s.GetSteamLibraries()
	if err != nil {
		return nil, fmt.Errorf("get installed Steam games: %w", err)
	}

	games, err := steam.ScanInstalledGames(libraries)
	if err != nil {
		return nil, fmt.Errorf("get installed Steam games: %w", err)
	}

	return games, nil
}

func (s *SteamService) GetStoredGames() (games []storage.StoredGame, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get stored games: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return nil, errors.New("storage is not configured")
	}

	return s.store.ListStoredGames(context.Background())
}

func (s *SteamService) GetGameImage(appID string, imageType steam.ImageType) (imageData string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get game image: %w", err)
		}
	}()

	appID = strings.TrimSpace(appID)
	if appID == "" {
		return "", fmt.Errorf("app ID is required")
	}

	paths, err := s.locateSteamInstallation()
	if err != nil {
		return "", err
	}

	imagePath, err := steam.ResolveGameImagePath(paths.Artwork, appID, imageType)
	if err != nil {
		return "", err
	}
	if imagePath == "" {
		return "", nil
	}

	content, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("read image: %w", err)
	}

	mimeType := http.DetectContentType(content)
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(content), nil
}

func (s *SteamService) ScanAndSaveSteamGames() (storage.SteamScanResult, error) {
	var result storage.SteamScanResult

	games, err := s.GetInstalledSteamGames()
	if err != nil {
		return result, fmt.Errorf("scan and save Steam games: %w", err)
	}

	result, err = s.store.SaveSteamScan(context.Background(), games)
	if err != nil {
		return storage.SteamScanResult{}, fmt.Errorf("scan and save Steam games: %w", err)
	}

	return result, nil
}

func (s *SteamService) locateSteamInstallation() (*steam.SteamPaths, error) {
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
