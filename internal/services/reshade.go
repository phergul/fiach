package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/phergul/mod-manager/internal/reshade"
	"github.com/phergul/mod-manager/internal/services/dto"
	"github.com/phergul/mod-manager/internal/services/dto/mappers"
	"github.com/phergul/mod-manager/internal/storage"
)

type ReshadeService struct {
	store           *storage.Store
	operatingSystem string
}

func NewReshadeService(store *storage.Store) *ReshadeService {
	return &ReshadeService{
		store:           store,
		operatingSystem: runtime.GOOS,
	}
}

func (s *ReshadeService) DetectGameReShade(ctx context.Context, gameID int64) (result dto.ReShadeDetectionResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("detect game ReShade runtime: %w", err)
		}
	}()

	if s.operatingSystem != "windows" {
		reason := "ReShade runtime detection is only supported on Windows."
		return dto.ReShadeDetectionResult{
			Status:            dto.ReShadeDetectionStatusUnsupported,
			Targets:           []dto.ReShadeTarget{},
			UnsupportedReason: &reason,
		}, nil
	}

	game, err := s.store.GetStoredGame(ctx, gameID)
	if err != nil {
		return dto.ReShadeDetectionResult{}, err
	}

	installPath := strings.TrimSpace(game.InstallPath)
	if installPath == "" {
		return dto.ReShadeDetectionResult{}, errors.New("game install path is required")
	}

	info, err := os.Stat(installPath)
	if err != nil {
		return dto.ReShadeDetectionResult{}, fmt.Errorf("inspect game install path: %w", err)
	}
	if !info.IsDir() {
		return dto.ReShadeDetectionResult{}, fmt.Errorf("game install path %q is not a directory", installPath)
	}

	scanResult, err := reshade.Scan(installPath)
	if err != nil {
		return dto.ReShadeDetectionResult{}, err
	}

	status := dto.ReShadeDetectionStatusNotInstalled
	if len(scanResult.Targets) > 0 {
		status = dto.ReShadeDetectionStatusInstalled
	}

	return dto.ReShadeDetectionResult{
		Status:  status,
		Targets: mappers.ToDTOReShadeTargets(scanResult.Targets),
	}, nil
}

func (s *ReshadeService) DownloadAndOpenReShadeInstaller(ctx context.Context) (result dto.ReShadeInstallerLaunchResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("download and open ReShade installer: %w", err)
		}
	}()

	if s.operatingSystem != "windows" {
		return dto.ReShadeInstallerLaunchResult{}, errors.New("ReShade installer launch is only supported on Windows")
	}

	launchResult, err := reshade.DownloadAndOpenInstaller(ctx, reshade.InstallerOptions{})
	if err != nil {
		return dto.ReShadeInstallerLaunchResult{}, err
	}

	return dto.ReShadeInstallerLaunchResult{
		Version: launchResult.Version,
	}, nil
}
