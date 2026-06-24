package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/injection"
	"github.com/phergul/fiach/internal/optiscaler"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
	"github.com/phergul/fiach/internal/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type OptiScalerService struct {
	store           *storage.Store
	manager         *optiscaler.Manager
	injection       *injection.Coordinator
	logger          *slog.Logger
	operatingSystem string
}

func NewOptiScalerService(store *storage.Store, logger *slog.Logger, coordinator *injection.Coordinator) *OptiScalerService {
	if logger == nil {
		logger = slog.Default()
	}
	if coordinator == nil {
		coordinator = injection.NewCoordinator(store)
	}
	return &OptiScalerService{
		store:           store,
		manager:         optiscaler.NewManager(store, optiscaler.ManagerOptions{}),
		injection:       coordinator,
		logger:          logger,
		operatingSystem: runtime.GOOS,
	}
}

func (s *OptiScalerService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) (err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationOptiScalerStartup, "OptiScaler startup recovery inspection started")
	defer func() {
		if err != nil {
			diag.fail("OptiScaler startup recovery inspection failed", err)
			err = fmt.Errorf("inspect OptiScaler recovery state at startup: %w", err)
		}
	}()
	state, err := s.manager.RecoveryState()
	if err != nil {
		return err
	}
	if state.Required {
		attrs := []slog.Attr{
			slog.String("operation", diagnostics.OperationOptiScalerRecovery),
			slog.String("journal_id", state.JournalID),
			slog.Int64("game_id", state.GameID),
			slog.String("target_path", state.TargetPath),
			slog.String("action", string(state.Action)),
			slog.Time("started_at", state.StartedAt),
			slog.String("recovery_error", state.Error),
		}
		if game, err := s.store.GetStoredGame(ctx, state.GameID); err == nil {
			attrs = append(attrs, slog.String("game_name", game.Name))
		}
		s.logger.LogAttrs(ctx, slog.LevelWarn, "OptiScaler recovery required", attrs...)
	}
	diag.complete("OptiScaler startup recovery inspection completed", optiScalerRecoveryStateAttrs(state)...)
	return nil
}

func (s *OptiScalerService) DiscoverOptiScalerCandidates(ctx context.Context, gameID int64) (result []dto.OptiScalerCandidate, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationDiscoverOptiScaler, "OptiScaler discovery started",
		slog.Int64("game_id", gameID),
		slog.String("operating_system", s.operatingSystem),
	)
	defer func() {
		if err != nil {
			diag.fail("OptiScaler discovery failed", err)
			err = fmt.Errorf("discover game OptiScaler candidates: %w", err)
		}
	}()
	if err := s.requireWindows(); err != nil {
		return nil, err
	}
	game, err := s.store.GetStoredGame(ctx, gameID)
	if err != nil {
		return nil, err
	}
	diag.attrs = append(diag.attrs, slog.String("game_name", game.Name))
	result, err = s.manager.Discover(ctx, game.InstallPath, gameID)
	if err == nil {
		diag.complete("OptiScaler discovery completed",
			diagnostics.PathAttr("install_path", game.InstallPath),
			slog.Int("candidate_count", len(result)),
		)
	}
	return result, err
}

func (s *OptiScalerService) ListOptiScalerTargets(ctx context.Context, gameID int64) (result []dto.OptiScalerTarget, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationListOptiScalerTargets, "OptiScaler targets list started",
		slog.Int64("game_id", gameID),
	)
	defer func() {
		if err != nil {
			diag.fail("OptiScaler targets list failed", err)
			err = fmt.Errorf("list game OptiScaler targets: %w", err)
		}
	}()
	targets, err := s.store.ListOptiScalerTargets(ctx, gameID)
	if err != nil {
		return nil, err
	}
	result = make([]dto.OptiScalerTarget, 0, len(targets))
	for _, target := range targets {
		result = append(result, mappers.ToDTOOptiScalerTarget(target))
	}
	diag.complete("OptiScaler targets list completed",
		slog.Int("target_count", len(result)),
	)
	return result, nil
}

func (s *OptiScalerService) GetOptiScalerReleaseStatus(ctx context.Context, refresh bool) (result dto.OptiScalerRelease, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationOptiScalerReleaseStatus, "OptiScaler release status started",
		slog.Bool("refresh", refresh),
		slog.String("operating_system", s.operatingSystem),
	)
	defer func() {
		if err != nil {
			diag.fail("OptiScaler release status failed", err)
			err = fmt.Errorf("get OptiScaler stable release status: %w", err)
		}
	}()
	if err := s.requireWindows(); err != nil {
		return dto.OptiScalerRelease{}, err
	}
	result, err = s.manager.StableRelease(ctx, refresh)
	if err != nil {
		diag.complete("OptiScaler release status completed with release error",
			diagnostics.ErrorAttr(err),
		)
		return dto.OptiScalerRelease{
			Error: optiScalerReleaseStatusError(err),
		}, nil
	}
	diag.complete("OptiScaler release status completed",
		slog.String("tag", result.Tag),
		slog.String("version", result.Version),
		slog.String("asset_name", result.AssetName),
		slog.Int64("size_bytes", result.Size),
	)
	return result, nil
}

func optiScalerReleaseStatusError(err error) string {
	return err.Error()
}

func (s *OptiScalerService) PreviewOptiScalerAction(ctx context.Context, request dto.OptiScalerRequest) (result dto.OptiScalerPreview, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationPreviewOptiScaler, "OptiScaler preview started",
		append(optiScalerRequestAttrs(request), slog.String("operating_system", s.operatingSystem))...,
	)
	defer func() {
		if err != nil {
			diag.fail("OptiScaler preview failed", err)
			err = fmt.Errorf("preview game OptiScaler action: %w", err)
		}
	}()
	if err := s.requireWindows(); err != nil {
		return dto.OptiScalerPreview{}, err
	}
	game, err := s.store.GetStoredGame(ctx, request.GameID)
	if err != nil {
		return dto.OptiScalerPreview{}, err
	}
	diag.attrs = append(diag.attrs, slog.String("game_name", game.Name))
	result, err = s.manager.Preview(ctx, game.InstallPath, request)
	if err == nil {
		diag.complete("OptiScaler preview completed",
			diagnostics.PathAttr("install_path", game.InstallPath),
			slog.Bool("can_apply", result.CanApply),
			slog.Int("operation_count", len(result.Operations)),
			slog.Int("configuration_change_count", len(result.ConfigurationChanges)),
			slog.Int("warning_count", len(result.Warnings)),
			slog.Int("conflict_count", len(result.Conflicts)),
			slog.Int("drift_count", len(result.Drift)),
			slog.String("release_tag", result.Release.Tag),
			slog.String("release_version", result.Release.Version),
			slog.String("release_asset_name", result.Release.AssetName),
		)
	}
	return result, err
}

func (s *OptiScalerService) ApplyOptiScalerAction(ctx context.Context, request dto.OptiScalerRequest, previewHash string) (result dto.OptiScalerApplyResult, err error) {
	attrs := append(optiScalerRequestAttrs(request),
		slog.Bool("preview_hash_provided", previewHash != ""),
		slog.String("operating_system", s.operatingSystem),
	)
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationApplyOptiScaler, "OptiScaler apply started", attrs...)
	defer func() {
		if err != nil {
			diag.fail("OptiScaler apply failed", err)
			err = fmt.Errorf("apply game OptiScaler action: %w", err)
		}
	}()
	if err := s.requireWindows(); err != nil {
		return dto.OptiScalerApplyResult{}, err
	}
	game, err := s.store.GetStoredGame(ctx, request.GameID)
	if err != nil {
		return dto.OptiScalerApplyResult{}, err
	}
	diag.attrs = append(diag.attrs, slog.String("game_name", game.Name))
	result, err = s.manager.Apply(ctx, game.InstallPath, request, previewHash)
	if err == nil {
		diag.complete("OptiScaler apply completed",
			diagnostics.PathAttr("install_path", game.InstallPath),
			slog.Bool("success", result.Success),
			slog.Bool("rolled_back", result.RolledBack),
			slog.String("message", result.Message),
		)
	}
	return result, err
}

func (s *OptiScalerService) GetOptiScalerRecoveryState(ctx context.Context) (result dto.OptiScalerRecoveryState, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationOptiScalerRecovery, "OptiScaler recovery state read started")
	defer func() {
		if err != nil {
			diag.fail("OptiScaler recovery state read failed", err)
			err = fmt.Errorf("get OptiScaler recovery state: %w", err)
		}
	}()
	result, err = s.manager.RecoveryState()
	if err != nil {
		return dto.OptiScalerRecoveryState{}, err
	}
	diag.complete("OptiScaler recovery state read completed", optiScalerRecoveryStateAttrs(result)...)
	return result, nil
}

func (s *OptiScalerService) RollbackOptiScalerRecovery(ctx context.Context, journalID string) (result dto.OptiScalerApplyResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationOptiScalerRecovery, "OptiScaler recovery rollback started",
		slog.String("journal_id", journalID),
	)
	defer func() {
		if err != nil {
			diag.fail("OptiScaler recovery rollback failed", err)
			err = fmt.Errorf("rollback OptiScaler recovery state: %w", err)
		}
	}()
	result, err = s.manager.RollbackRecovery(journalID)
	if err != nil {
		return dto.OptiScalerApplyResult{}, err
	}
	diag.complete("OptiScaler recovery rollback completed",
		slog.Bool("success", result.Success),
		slog.Bool("rolled_back", result.RolledBack),
		slog.String("message", result.Message),
	)
	return result, nil
}

func (s *OptiScalerService) requireWindows() error {
	if s.operatingSystem != "windows" {
		return errors.New("OptiScaler management is only supported on Windows")
	}
	return nil
}

func optiScalerRequestAttrs(request dto.OptiScalerRequest) []slog.Attr {
	attrs := []slog.Attr{
		slog.Int64("game_id", request.GameID),
		slog.String("action", string(request.Action)),
		slog.String("target_relative_path", request.TargetRelativePath),
		slog.String("executable_relative_path", request.ExecutableRelativePath),
		slog.String("graphics_api", string(request.GraphicsAPI)),
		slog.String("proxy_filename", request.ProxyFilename),
		slog.Bool("dxgi_spoofing", request.DXGISpoofing),
		slog.Bool("acknowledge_warning", request.AcknowledgeWarning),
		slog.Bool("backup_and_continue", request.BackupAndContinue),
		slog.Bool("enable_reshade_coexistence", request.EnableReShadeCoexistence),
	}
	if request.ProcessFilter != nil {
		attrs = append(attrs, slog.String("process_filter", *request.ProcessFilter))
	}
	return attrs
}

func optiScalerRecoveryStateAttrs(state dto.OptiScalerRecoveryState) []slog.Attr {
	return []slog.Attr{
		slog.Bool("required", state.Required),
		slog.String("journal_id", state.JournalID),
		slog.Int64("game_id", state.GameID),
		slog.String("target_path", state.TargetPath),
		slog.String("action", string(state.Action)),
		slog.Time("started_at", state.StartedAt),
		slog.String("recovery_error", state.Error),
	}
}
