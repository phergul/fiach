package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/injection"
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
	injection       *injection.Coordinator
}

func NewReshadeService(store *storage.Store, logger *slog.Logger, coordinator *injection.Coordinator) *ReshadeService {
	if logger == nil {
		logger = slog.Default()
	}
	if coordinator == nil {
		coordinator = injection.NewCoordinator(store)
	}

	return &ReshadeService{
		store:           store,
		logger:          logger,
		operatingSystem: runtime.GOOS,
		scan:            reshade.ScanManaged,
		manager:         reshade.NewManager(store, reshade.ManagerOptions{}),
		injection:       coordinator,
	}
}

func (s *ReshadeService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) (err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationReShadeStartup, "ReShade startup recovery inspection started")
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("ReShade startup recovery inspection failed", err, platformUserError)
		}
	}()
	state, err := s.manager.RecoveryState()
	if err != nil {
		return err
	}
	if state.Required {
		attrs := []slog.Attr{
			slog.String("journal_id", state.JournalID),
			slog.Int64("game_id", state.GameID),
			slog.String("target_path", state.TargetPath),
			slog.String("action", string(state.Action)),
			slog.Time("started_at", state.StartedAt),
			slog.String("recovery_error", state.Error),
		}
		if game, lookupErr := s.store.GetStoredGame(ctx, state.GameID); lookupErr == nil {
			attrs = append(attrs, slog.String("game_name", game.Name))
		}
		logOperationEvent(ctx, s.logger, slog.LevelWarn, diagnostics.OperationReShadeRecovery, "recovery_required", "ReShade recovery required", attrs...)
	}
	diag.complete("ReShade startup recovery inspection completed", reShadeRecoveryStateAttrs(ctx, s.store, state)...)
	return nil
}

func (s *ReshadeService) DetectGameReShade(ctx context.Context, gameID int64) (result dto.ReShadeDetectionResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationDetectReShade, "ReShade detection started",
		slog.Int64("game_id", gameID),
		slog.String("operating_system", s.operatingSystem),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("ReShade detection failed", err, platformUserError)
		}
	}()

	if s.operatingSystem != "windows" {
		reason := "ReShade runtime detection is only supported on Windows."
		result = dto.ReShadeDetectionResult{
			Status:            dto.ReShadeDetectionStatusUnsupported,
			Targets:           []dto.DetectedReShadeTarget{},
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
	diag.attrs = append(diag.attrs, slog.String("game_name", game.Name))

	installPath := strings.TrimSpace(game.InstallPath)
	if installPath == "" {
		return dto.ReShadeDetectionResult{}, apperror.New("This game does not have an install path configured.")
	}

	info, err := os.Stat(installPath)
	if err != nil {
		return dto.ReShadeDetectionResult{}, fmt.Errorf("inspect game install path: %w", err)
	}
	if !info.IsDir() {
		return dto.ReShadeDetectionResult{}, fmt.Errorf("game install path %q is not a directory", installPath)
	}

	managedTargets, err := s.injection.ListManagedOptiScalerTargets(ctx, gameID)
	if err != nil {
		return dto.ReShadeDetectionResult{}, err
	}
	managedChainedTargets := make([]string, 0, len(managedTargets))
	for _, target := range managedTargets {
		if target.GraphicsAPI != "directx" {
			continue
		}
		path, resolveErr := reshade.ResolveWithinRoot(installPath, target.TargetRelativePath)
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
		detectedTargets = append(detectedTargets, dto.DetectedReShadeTarget{
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
		slog.Int("managed_chain_target_count", len(managedChainedTargets)),
		slog.Int("managed_target_count", len(managedReShadeTargets)),
		diagnostics.PathAttr("install_path", installPath),
	)

	return result, nil
}

func (s *ReshadeService) ListReShadeTargets(ctx context.Context, gameID int64) (result []dto.ReShadeTarget, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationListReShadeTargets, "ReShade targets list started",
		slog.Int64("game_id", gameID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("ReShade targets list failed", err, platformUserError)
		}
	}()
	game, err := s.store.GetStoredGame(ctx, gameID)
	if err != nil {
		return nil, err
	}
	diag.attrs = append(diag.attrs, slog.String("game_name", game.Name))
	result, err = s.manager.ListTargets(ctx, game.InstallPath, gameID)
	if err != nil {
		return nil, err
	}
	diag.complete("ReShade targets list completed",
		diagnostics.PathAttr("install_path", game.InstallPath),
		slog.Int("target_count", len(result)),
	)
	return result, nil
}

func (s *ReshadeService) ListReShadeContentCatalogue(
	ctx context.Context,
	refresh bool,
) (result dto.ReShadeContentCatalogue, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationListReShadeContent, "ReShade content catalogue list started",
		slog.Bool("refresh", refresh),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("ReShade content catalogue list failed", err, platformUserError)
		}
	}()
	result, err = s.manager.ListContentCatalogue(ctx, refresh)
	if err != nil {
		return dto.ReShadeContentCatalogue{}, err
	}
	diag.complete("ReShade content catalogue list completed",
		slog.Bool("cached", result.Cached),
		slog.Int("effect_package_count", len(result.Effects)),
		slog.Int("addon_count", len(result.Addons)),
	)
	return result, nil
}

func (s *ReshadeService) GetReShadeInstallerStatus(
	ctx context.Context,
	refresh bool,
) (result dto.ReShadeInstallerStatus, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationReShadeInstallerStatus, "ReShade installer status started",
		slog.Bool("refresh", refresh),
		slog.String("operating_system", s.operatingSystem),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("ReShade installer status failed", err, platformUserError)
		}
	}()
	if s.operatingSystem != "windows" {
		return dto.ReShadeInstallerStatus{}, apperror.New("ReShade management is only supported on Windows.")
	}
	result = reshade.ResolveInstallerStatus(ctx, refresh)
	diag.complete("ReShade installer status completed",
		slog.String("standard_version", result.Standard.Version),
		slog.String("standard_asset_name", result.Standard.AssetName),
		slog.Bool("standard_cached", result.Standard.Cached),
		slog.String("standard_error", result.Standard.Error),
		slog.String("addon_version", result.Addon.Version),
		slog.String("addon_asset_name", result.Addon.AssetName),
		slog.Bool("addon_cached", result.Addon.Cached),
		slog.String("addon_error", result.Addon.Error),
	)
	return result, nil
}

func (s *ReshadeService) ListReShadeChainTargets(
	ctx context.Context,
	gameID int64,
) (result []dto.ReShadeChainTarget, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationListReShadeChainTargets, "ReShade chain targets list started",
		slog.Int64("game_id", gameID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("ReShade chain targets list failed", err, platformUserError)
		}
	}()
	if game, lookupErr := s.store.GetStoredGame(ctx, gameID); lookupErr == nil {
		diag.attrs = append(diag.attrs, slog.String("game_name", game.Name))
	}
	targets, err := s.injection.ListTargets(ctx, gameID)
	if err != nil {
		return nil, err
	}
	result = make([]dto.ReShadeChainTarget, 0, len(targets))
	for _, target := range targets {
		result = append(result, mappers.ToDTOReShadeChainTarget(target))
	}
	diag.complete("ReShade chain targets list completed",
		slog.Int("target_count", len(result)),
	)
	return result, nil
}

func (s *ReshadeService) InspectReShadePreset(
	ctx context.Context,
	gameID int64,
	targetRelativePath string,
	presetPath string,
) (result dto.ReShadePresetInspectionResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationInspectReShadePreset, "ReShade preset inspection started",
		slog.Int64("game_id", gameID),
		slog.String("target_relative_path", targetRelativePath),
		diagnostics.PathAttr("preset_path", presetPath),
		slog.Bool("preset_path_absolute", filepath.IsAbs(presetPath)),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("ReShade preset inspection failed", err, platformUserError)
		}
	}()
	game, err := s.store.GetStoredGame(ctx, gameID)
	if err != nil {
		return dto.ReShadePresetInspectionResult{}, err
	}
	diag.attrs = append(diag.attrs, slog.String("game_name", game.Name))
	targetPath, err := reshade.ResolveWithinRoot(game.InstallPath, targetRelativePath)
	if err != nil {
		return dto.ReShadePresetInspectionResult{}, err
	}
	resolvedPresetPath := presetPath
	if !filepath.IsAbs(resolvedPresetPath) {
		resolvedPresetPath = filepath.Join(targetPath, presetPath)
	}
	if err := fileops.RequirePathWithinRoot("ReShade preset", resolvedPresetPath, game.InstallPath); err != nil {
		return dto.ReShadePresetInspectionResult{}, err
	}
	catalogue, err := s.manager.ListContentCatalogue(ctx, false)
	if err != nil {
		return dto.ReShadePresetInspectionResult{}, err
	}
	result, err = reshade.InspectPreset(resolvedPresetPath, catalogue)
	if err != nil {
		return dto.ReShadePresetInspectionResult{}, err
	}
	diag.complete("ReShade preset inspection completed",
		diagnostics.PathAttr("install_path", game.InstallPath),
		diagnostics.PathAttr("target_path", targetPath),
		diagnostics.PathAttr("resolved_preset_path", resolvedPresetPath),
		slog.Int("referenced_effect_count", len(result.ReferencedEffects)),
		slog.Int("recommendation_count", len(result.Recommendations)),
		slog.Int("missing_effect_count", len(result.MissingEffects)),
		slog.Int("warning_count", len(result.Warnings)),
	)
	return result, nil
}

func (s *ReshadeService) DiscoverReShadeCandidates(
	ctx context.Context,
	gameID int64,
) (result dto.ReShadeDiscoveryResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationDiscoverReShade, "ReShade candidate discovery started",
		slog.Int64("game_id", gameID),
		slog.String("operating_system", s.operatingSystem),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("ReShade candidate discovery failed", err, platformUserError)
		}
	}()
	if s.operatingSystem != "windows" {
		return dto.ReShadeDiscoveryResult{}, apperror.New(
			"ReShade discovery is only supported on Windows.",
		)
	}
	game, err := s.store.GetStoredGame(ctx, gameID)
	if err != nil {
		return dto.ReShadeDiscoveryResult{}, err
	}
	diag.attrs = append(diag.attrs, slog.String("game_name", game.Name))
	optiScalerTargets, _, err := s.injection.ManagedDirectXOptiScalerTargets(ctx, gameID)
	if err != nil {
		return dto.ReShadeDiscoveryResult{}, err
	}
	allowedProxyPaths := make([]string, 0, len(optiScalerTargets))
	for _, target := range optiScalerTargets {
		targetPath, resolveErr := reshade.ResolveWithinRoot(game.InstallPath, target.TargetRelativePath)
		if resolveErr != nil {
			return dto.ReShadeDiscoveryResult{}, resolveErr
		}
		allowedProxyPaths = append(allowedProxyPaths, filepath.Join(targetPath, target.ProxyFilename))
	}
	result, err = reshade.DiscoverCandidates(game.InstallPath, reshade.DiscoveryOptions{
		AllowedForeignProxyPaths: allowedProxyPaths,
	})
	if err != nil {
		return dto.ReShadeDiscoveryResult{}, err
	}
	diag.complete("ReShade candidate discovery completed",
		diagnostics.PathAttr("install_path", game.InstallPath),
		slog.Int("candidate_count", len(result.Candidates)),
		slog.Int("allowed_optiscaler_proxy_count", len(allowedProxyPaths)),
		slog.Int("warning_count", len(result.Warnings)),
	)
	return result, nil
}

func (s *ReshadeService) PreviewReShadeAction(ctx context.Context, request dto.ReShadeRequest) (result dto.ReShadePreview, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationPreviewReShade, "ReShade preview started",
		append(reShadeRequestAttrs(request), slog.String("operating_system", s.operatingSystem))...,
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("ReShade preview failed", err, platformUserError)
		}
	}()
	if s.operatingSystem != "windows" {
		return dto.ReShadePreview{}, apperror.New("ReShade management is only supported on Windows.")
	}
	game, err := s.store.GetStoredGame(ctx, request.GameID)
	if err != nil {
		return dto.ReShadePreview{}, err
	}
	diag.attrs = append(diag.attrs, slog.String("game_name", game.Name))
	result, err = s.manager.Preview(ctx, game.InstallPath, request)
	if err != nil {
		return dto.ReShadePreview{}, err
	}
	diag.complete("ReShade preview completed",
		diagnostics.PathAttr("install_path", game.InstallPath),
		slog.Bool("can_apply", result.CanApply),
		slog.Int("operation_count", len(result.Operations)),
		slog.Int("path_impact_count", len(result.PathImpacts)),
		slog.Int("warning_count", len(result.Warnings)),
		slog.Int("conflict_count", len(result.Conflicts)),
		slog.Int("drift_count", len(result.Drift)),
		slog.Int("user_content_drift_count", len(result.UserContentDrift)),
		slog.Bool("desired_target_present", result.DesiredTarget != nil),
	)
	return result, nil
}

func (s *ReshadeService) ApplyReShadeAction(
	ctx context.Context,
	request dto.ReShadeRequest,
	previewHash string,
) (result dto.ReShadeApplyResult, err error) {
	attrs := append(reShadeRequestAttrs(request),
		slog.Bool("preview_hash_provided", previewHash != ""),
		slog.String("operating_system", s.operatingSystem),
	)
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationApplyReShade, "ReShade apply started", attrs...)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("ReShade apply failed", err, platformUserError)
		}
	}()
	if s.operatingSystem != "windows" {
		return dto.ReShadeApplyResult{}, apperror.New("ReShade management is only supported on Windows.")
	}
	game, err := s.store.GetStoredGame(ctx, request.GameID)
	if err != nil {
		return dto.ReShadeApplyResult{}, err
	}
	diag.attrs = append(diag.attrs, slog.String("game_name", game.Name))
	result, err = s.manager.Apply(ctx, game.InstallPath, request, previewHash)
	if err != nil {
		return dto.ReShadeApplyResult{}, err
	}
	diag.complete("ReShade apply completed",
		diagnostics.PathAttr("install_path", game.InstallPath),
		slog.Bool("success", result.Success),
		slog.Bool("rolled_back", result.RolledBack),
		slog.String("message", result.Message),
	)
	return result, nil
}

func (s *ReshadeService) GetReShadeRecoveryState(ctx context.Context) (result dto.ReShadeRecoveryState, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationReShadeRecovery, "ReShade recovery state read started")
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("ReShade recovery state read failed", err, platformUserError)
		}
	}()
	result, err = s.manager.RecoveryState()
	if err != nil {
		return dto.ReShadeRecoveryState{}, err
	}
	if result.GameID > 0 {
		if game, lookupErr := s.store.GetStoredGame(ctx, result.GameID); lookupErr == nil {
			diag.attrs = append(diag.attrs, slog.String("game_name", game.Name))
		}
	}
	diag.complete("ReShade recovery state read completed", reShadeRecoveryStateAttrs(ctx, s.store, result)...)
	return result, nil
}

func (s *ReshadeService) RollbackReShadeRecovery(
	ctx context.Context,
	journalID string,
) (result dto.ReShadeApplyResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationReShadeRecovery, "ReShade recovery rollback started",
		slog.String("journal_id", journalID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("ReShade recovery rollback failed", err, platformUserError)
		}
	}()
	if state, stateErr := s.manager.RecoveryState(); stateErr == nil && state.JournalID == journalID && state.GameID > 0 {
		diag.attrs = append(diag.attrs, slog.Int64("game_id", state.GameID))
		if game, lookupErr := s.store.GetStoredGame(ctx, state.GameID); lookupErr == nil {
			diag.attrs = append(diag.attrs, slog.String("game_name", game.Name))
		}
	}
	result, err = s.manager.RollbackRecovery(journalID)
	if err != nil {
		return dto.ReShadeApplyResult{}, err
	}
	diag.complete("ReShade recovery rollback completed",
		slog.Bool("success", result.Success),
		slog.Bool("rolled_back", result.RolledBack),
		slog.String("message", result.Message),
	)
	return result, nil
}

func reShadeRequestAttrs(request dto.ReShadeRequest) []slog.Attr {
	return []slog.Attr{
		slog.Int64("game_id", request.GameID),
		slog.String("action", string(request.Action)),
		slog.String("target_relative_path", request.TargetRelativePath),
		slog.String("executable_relative_path", request.ExecutableRelativePath),
		slog.String("rendering_api", string(request.RenderingAPI)),
		slog.String("proxy_filename", request.ProxyFilename),
		slog.String("architecture", string(request.Architecture)),
		slog.String("build_variant", string(request.BuildVariant)),
		slog.Bool("backup_and_continue", request.BackupAndContinue),
		slog.Bool("single_player_acknowledged", request.SinglePlayerAcknowledged),
		slog.Bool("anti_cheat_risk_acknowledged", request.AntiCheatRiskAcknowledged),
		slog.Int("effect_package_count", len(request.Content.EffectPackages)),
		slog.Int("addon_count", len(request.Content.Addons)),
	}
}

func reShadeRecoveryStateAttrs(ctx context.Context, store *storage.Store, state dto.ReShadeRecoveryState) []slog.Attr {
	attrs := []slog.Attr{
		slog.Bool("required", state.Required),
		slog.String("journal_id", state.JournalID),
		slog.Int64("game_id", state.GameID),
		slog.String("target_path", state.TargetPath),
		slog.String("action", string(state.Action)),
		slog.Time("started_at", state.StartedAt),
		slog.String("recovery_error", state.Error),
	}
	if store != nil && state.GameID > 0 {
		if game, err := store.GetStoredGame(ctx, state.GameID); err == nil {
			attrs = append(attrs, slog.String("game_name", game.Name))
		}
	}
	return attrs
}
