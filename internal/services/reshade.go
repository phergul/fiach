package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/optiscaler"
	"github.com/phergul/fiach/internal/reshade"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
	"github.com/phergul/fiach/internal/storage"
)

type ReshadeService struct {
	store           *storage.Store
	logger          *slog.Logger
	operatingSystem string
	scan            func(string, []string) (reshade.Result, error)
}

func NewReshadeService(store *storage.Store, logger *slog.Logger) *ReshadeService {
	if logger == nil {
		logger = slog.Default()
	}

	return &ReshadeService{
		store:           store,
		logger:          logger,
		operatingSystem: runtime.GOOS,
		scan:            reshade.ScanManaged,
	}
}

func (s *ReshadeService) DetectGameReShade(ctx context.Context, gameID int64) (result dto.ReShadeDetectionResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationDetectReShade, "ReShade detection started",
		slog.Int64("game_id", gameID),
	)
	defer func() {
		if err != nil {
			diag.fail("ReShade detection failed", err)
			err = fmt.Errorf("detect game ReShade runtime: %w", err)
		}
	}()

	if s.operatingSystem != "windows" {
		reason := "ReShade runtime detection is only supported on Windows."
		result = dto.ReShadeDetectionResult{
			Status:            dto.ReShadeDetectionStatusUnsupported,
			Targets:           []dto.ReShadeTarget{},
			UnsupportedReason: &reason,
		}
		diag.complete("ReShade detection completed",
			slog.String("status", string(result.Status)),
		)
		return result, nil
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

	managedTargets, err := s.store.ListOptiScalerTargets(ctx, gameID)
	if err != nil {
		return dto.ReShadeDetectionResult{}, err
	}
	managedChainedTargets := make([]string, 0, len(managedTargets))
	for _, target := range managedTargets {
		if target.GraphicsAPI != string(optiscaler.GraphicsAPIDirectX) {
			continue
		}
		path, resolveErr := optiscaler.ResolveWithinRoot(installPath, target.TargetRelativePath)
		if resolveErr != nil {
			return dto.ReShadeDetectionResult{}, resolveErr
		}
		managedChainedTargets = append(managedChainedTargets, path)
	}
	scanResult, err := s.scan(installPath, managedChainedTargets)
	if err != nil {
		return dto.ReShadeDetectionResult{}, err
	}

	status := dto.ReShadeDetectionStatusNotInstalled
	if len(scanResult.Targets) > 0 {
		status = dto.ReShadeDetectionStatusInstalled
	}

	result = dto.ReShadeDetectionResult{
		Status:  status,
		Targets: mappers.ToDTOReShadeTargets(scanResult.Targets),
	}
	diag.complete("ReShade detection completed",
		slog.String("status", string(result.Status)),
		slog.Int("target_count", len(result.Targets)),
	)

	return result, nil
}

func (s *ReshadeService) PreflightReShadeInstaller(
	ctx context.Context,
	gameID int64,
	variant optiscaler.ReShadeInstallerVariant,
) (result dto.ReShadeInstallerPreflight, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("preflight game ReShade installer: %w", err)
		}
	}()
	if s.operatingSystem != "windows" {
		return dto.ReShadeInstallerPreflight{}, errors.New("ReShade installer launch is only supported on Windows")
	}
	if variant != optiscaler.ReShadeInstallerVariantStandard &&
		variant != optiscaler.ReShadeInstallerVariantAddon {
		return dto.ReShadeInstallerPreflight{}, errors.New("installer variant is invalid")
	}
	targets, err := s.store.ListOptiScalerTargets(ctx, gameID)
	if err != nil {
		return dto.ReShadeInstallerPreflight{}, err
	}
	result = dto.ReShadeInstallerPreflight{
		Disposition: dto.ReShadeInstallerPreflightOrdinary,
		Variant:     variant,
		Targets:     []dto.ReShadeManagedTarget{},
	}
	for _, target := range targets {
		if target.GraphicsAPI == string(optiscaler.GraphicsAPIVulkan) {
			result.Disposition = dto.ReShadeInstallerPreflightBlocked
			result.Message = "Automated Vulkan and ReShade coexistence is not supported for managed OptiScaler targets."
			result.Targets = nil
			return result, nil
		}
		if target.GraphicsAPI == string(optiscaler.GraphicsAPIDirectX) {
			result.Targets = append(result.Targets, dto.ReShadeManagedTarget{
				TargetRelativePath:     target.TargetRelativePath,
				ExecutableRelativePath: target.ExecutableRelativePath,
				ProxyFilename:          target.ProxyFilename,
			})
		}
	}
	if len(result.Targets) > 0 {
		result.Disposition = dto.ReShadeInstallerPreflightCoordinated
		result.Message = "Select the managed DirectX target to preserve the OptiScaler chain."
	}
	return result, nil
}

func (s *ReshadeService) DownloadAndOpenReShadeInstaller(ctx context.Context) (result dto.ReShadeInstallerLaunchResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationLaunchReShadeInstaller, "ReShade installer launch started",
		slog.String("installer_variant", "standard"),
	)
	defer func() {
		if err != nil {
			diag.fail("ReShade installer launch failed", err)
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

	result = dto.ReShadeInstallerLaunchResult{
		Version: launchResult.Version,
	}
	diag.complete("ReShade installer launch completed",
		slog.String("version", result.Version),
	)

	return result, nil
}

func (s *ReshadeService) DownloadAndOpenReShadeAddonInstaller(ctx context.Context) (result dto.ReShadeInstallerLaunchResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationLaunchReShadeInstaller, "ReShade add-on installer launch started",
		slog.String("installer_variant", "addon"),
	)
	defer func() {
		if err != nil {
			diag.fail("ReShade add-on installer launch failed", err)
			err = fmt.Errorf("download and open ReShade add-on installer: %w", err)
		}
	}()

	if s.operatingSystem != "windows" {
		return dto.ReShadeInstallerLaunchResult{}, errors.New("ReShade add-on installer launch is only supported on Windows")
	}

	launchResult, err := reshade.DownloadAndOpenInstaller(ctx, reshade.InstallerOptions{
		Variant: reshade.InstallerVariantAddon,
	})
	if err != nil {
		return dto.ReShadeInstallerLaunchResult{}, err
	}

	result = dto.ReShadeInstallerLaunchResult{
		Version: launchResult.Version,
	}
	diag.complete("ReShade add-on installer launch completed",
		slog.String("installer_variant", "addon"),
		slog.String("version", result.Version),
	)

	return result, nil
}
