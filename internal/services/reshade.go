package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/optiscaler"
	"github.com/phergul/fiach/internal/reshade"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
	"github.com/phergul/fiach/internal/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type ReshadeService struct {
	store           *storage.Store
	logger          *slog.Logger
	operatingSystem string
	scan            func(string, []string) (reshade.Result, error)
	manager         *reshade.Manager
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
		manager:         reshade.NewManager(store, reshade.ManagerOptions{}),
	}
}

func (s *ReshadeService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("inspect ReShade recovery state at startup: %w", err)
		}
	}()
	state, err := s.manager.RecoveryState()
	if err != nil {
		return err
	}
	if state.Required {
		s.logger.WarnContext(ctx, "ReShade recovery required",
			slog.String("journal_id", state.JournalID),
			slog.Int64("game_id", state.GameID),
		)
	}
	return nil
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

	managedReShadeTargets, err := s.manager.ListTargets(ctx, installPath, gameID)
	if err != nil {
		return dto.ReShadeDetectionResult{}, err
	}
	managedByPath := make(map[string]reshade.ManagementStatus, len(managedReShadeTargets))
	for _, target := range managedReShadeTargets {
		path, resolveErr := reshade.ResolveWithinRoot(installPath, target.TargetRelativePath)
		if resolveErr != nil {
			return dto.ReShadeDetectionResult{}, resolveErr
		}
		managedByPath[strings.ToLower(filepath.Clean(path))] = target.Status
	}
	detectedTargets := mappers.ToDTOReShadeTargets(scanResult.Targets)
	for index := range detectedTargets {
		if managedStatus, ok := managedByPath[strings.ToLower(filepath.Clean(detectedTargets[index].Path))]; ok {
			detectedTargets[index].ManagementStatus = managedStatus
			delete(managedByPath, strings.ToLower(filepath.Clean(detectedTargets[index].Path)))
		}
	}
	for _, target := range managedReShadeTargets {
		path, resolveErr := reshade.ResolveWithinRoot(installPath, target.TargetRelativePath)
		if resolveErr != nil {
			return dto.ReShadeDetectionResult{}, resolveErr
		}
		if _, missing := managedByPath[strings.ToLower(filepath.Clean(path))]; !missing {
			continue
		}
		executablePath, resolveErr := reshade.ResolveWithinRoot(installPath, target.ExecutableRelativePath)
		if resolveErr != nil {
			return dto.ReShadeDetectionResult{}, resolveErr
		}
		detectedTargets = append(detectedTargets, dto.ReShadeTarget{
			Path: path, Executables: []string{executablePath}, ManagementStatus: target.Status,
		})
	}
	if len(detectedTargets) > 0 {
		status = dto.ReShadeDetectionStatusInstalled
	}
	result = dto.ReShadeDetectionResult{
		Status:  status,
		Targets: detectedTargets,
	}
	diag.complete("ReShade detection completed",
		slog.String("status", string(result.Status)),
		slog.Int("target_count", len(result.Targets)),
	)

	return result, nil
}

func (s *ReshadeService) ListManagedReShadeTargets(ctx context.Context, gameID int64) (result []dto.ManagedReShadeTarget, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list game managed ReShade targets: %w", err)
		}
	}()
	game, err := s.store.GetStoredGame(ctx, gameID)
	if err != nil {
		return nil, err
	}
	return s.manager.ListTargets(ctx, game.InstallPath, gameID)
}

func (s *ReshadeService) PreviewManagedReShadeAction(ctx context.Context, request dto.ManagedReShadeRequest) (result dto.ManagedReShadePreview, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("preview game managed ReShade action: %w", err)
		}
	}()
	if s.operatingSystem != "windows" {
		return dto.ManagedReShadePreview{}, errors.New("managed ReShade is only supported on Windows")
	}
	game, err := s.store.GetStoredGame(ctx, request.GameID)
	if err != nil {
		return dto.ManagedReShadePreview{}, err
	}
	return s.manager.Preview(ctx, game.InstallPath, request)
}

func (s *ReshadeService) ApplyManagedReShadeAction(
	ctx context.Context,
	request dto.ManagedReShadeRequest,
	previewHash string,
) (result dto.ManagedReShadeApplyResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("apply game managed ReShade action: %w", err)
		}
	}()
	if s.operatingSystem != "windows" {
		return dto.ManagedReShadeApplyResult{}, errors.New("managed ReShade is only supported on Windows")
	}
	game, err := s.store.GetStoredGame(ctx, request.GameID)
	if err != nil {
		return dto.ManagedReShadeApplyResult{}, err
	}
	return s.manager.Apply(ctx, game.InstallPath, request, previewHash)
}

func (s *ReshadeService) GetManagedReShadeRecoveryState(_ context.Context) (result dto.ManagedReShadeRecoveryState, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get managed ReShade recovery state: %w", err)
		}
	}()
	return s.manager.RecoveryState()
}

func (s *ReshadeService) RollbackManagedReShadeRecovery(
	_ context.Context,
	journalID string,
) (result dto.ManagedReShadeApplyResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("rollback managed ReShade recovery state: %w", err)
		}
	}()
	return s.manager.RollbackRecovery(journalID)
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
