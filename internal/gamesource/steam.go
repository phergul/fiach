package gamesource

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/phergul/fiach/internal/steam"
	"github.com/phergul/fiach/internal/storage"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

const SteamInstallPathSettingKey = "steam.install_path"

type SteamSource struct {
	store         *storage.Store
	artworkRootMu sync.Mutex
	artworkRoot   string
}

func NewSteamSource(store *storage.Store) *SteamSource {
	return &SteamSource{
		store: store,
	}
}

func (s *SteamSource) Source() string {
	return dbtypes.GameSourceSteam
}

func (s *SteamSource) ScanGames(ctx context.Context) (games []dbtypes.SourceGame, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("scan Steam games: %w", err)
		}
	}()

	installed, err := s.getInstalledSteamGames(ctx)
	if err != nil {
		return nil, err
	}

	sourceGames := make([]dbtypes.SourceGame, 0, len(installed))
	for _, game := range installed {
		sourceGames = append(sourceGames, dbtypes.SourceGame{
			SourceID:    game.AppID,
			Name:        game.Name,
			InstallPath: game.InstallPath,
		})
	}

	return sourceGames, nil
}

func (s *SteamSource) getSteamLibraries(ctx context.Context) (libraries []string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get Steam libraries: %w", err)
		}
	}()

	paths, err := s.locateSteamInstallation(ctx)
	if err != nil {
		return nil, err
	}

	libraries, err = steam.ParseLibraryFolders(paths)
	if err != nil {
		return nil, err
	}

	return libraries, nil
}

func (s *SteamSource) getInstalledSteamGames(ctx context.Context) (games []steam.Game, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get installed Steam games: %w", err)
		}
	}()

	libraries, err := s.getSteamLibraries(ctx)
	if err != nil {
		return nil, err
	}

	games, err = steam.ScanInstalledGames(libraries)
	if err != nil {
		return nil, err
	}

	return games, nil
}

func (s *SteamSource) getArtworkRoot(ctx context.Context) (root string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("resolve Steam artwork root: %w", err)
		}
	}()

	s.artworkRootMu.Lock()
	defer s.artworkRootMu.Unlock()

	if s.artworkRoot != "" {
		return s.artworkRoot, nil
	}

	paths, err := s.locateSteamInstallation(ctx)
	if err != nil {
		return "", err
	}

	s.artworkRoot = paths.Artwork
	return s.artworkRoot, nil
}

func (s *SteamSource) locateSteamInstallation(ctx context.Context) (paths *steam.SteamPaths, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("locate Steam installation: %w", err)
		}
	}()

	manualPath, found, err := s.store.GetSetting(ctx, SteamInstallPathSettingKey)
	if err != nil {
		return nil, fmt.Errorf("read Steam installation setting: %w", err)
	}
	if !found {
		manualPath = ""
	}

	paths, err = steam.FindSteamPaths(manualPath)
	if err != nil {
		if errors.Is(err, steam.ErrSteamNotFound) {
			return nil, fmt.Errorf("Steam installation could not be found. Configure a valid Steam path in settings or install Steam in a standard location: %w", err)
		}

		return nil, fmt.Errorf("find Steam installation: %w", err)
	}

	return paths, nil
}
