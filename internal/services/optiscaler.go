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
	defer func() {
		if err != nil {
			err = fmt.Errorf("inspect OptiScaler recovery state at startup: %w", err)
		}
	}()
	state, err := s.manager.RecoveryState()
	if err != nil {
		return err
	}
	if state.Required {
		s.logger.WarnContext(ctx, "OptiScaler recovery required",
			slog.String("operation", diagnostics.OperationOptiScalerRecovery),
			slog.String("journal_id", state.JournalID),
			slog.Int64("game_id", state.GameID),
		)
	}
	return nil
}

func (s *OptiScalerService) DiscoverOptiScalerCandidates(ctx context.Context, gameID int64) (result []dto.OptiScalerCandidate, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationDiscoverOptiScaler, "OptiScaler discovery started", slog.Int64("game_id", gameID))
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
	result, err = s.manager.Discover(ctx, game.InstallPath, gameID)
	if err == nil {
		diag.complete("OptiScaler discovery completed", slog.Int("candidate_count", len(result)))
	}
	return result, err
}

func (s *OptiScalerService) ListOptiScalerTargets(ctx context.Context, gameID int64) (result []dto.OptiScalerTarget, err error) {
	defer func() {
		if err != nil {
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
	return result, nil
}

func (s *OptiScalerService) GetOptiScalerReleaseStatus(ctx context.Context) (result dto.OptiScalerRelease, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get OptiScaler stable release status: %w", err)
		}
	}()
	if err := s.requireWindows(); err != nil {
		return dto.OptiScalerRelease{}, err
	}
	return s.manager.StableRelease(ctx)
}

func (s *OptiScalerService) PreviewOptiScalerAction(ctx context.Context, request dto.OptiScalerRequest) (result dto.OptiScalerPreview, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationPreviewOptiScaler, "OptiScaler preview started",
		slog.Int64("game_id", request.GameID), slog.String("action", string(request.Action)))
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
	result, err = s.manager.Preview(ctx, game.InstallPath, request)
	if err == nil {
		diag.complete("OptiScaler preview completed", slog.Bool("can_apply", result.CanApply), slog.Int("conflict_count", len(result.Conflicts)))
	}
	return result, err
}

func (s *OptiScalerService) ApplyOptiScalerAction(ctx context.Context, request dto.OptiScalerRequest, previewHash string) (result dto.OptiScalerApplyResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationApplyOptiScaler, "OptiScaler apply started",
		slog.Int64("game_id", request.GameID), slog.String("action", string(request.Action)))
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
	result, err = s.manager.Apply(ctx, game.InstallPath, request, previewHash)
	if err == nil {
		diag.complete("OptiScaler apply completed", slog.Bool("success", result.Success))
	}
	return result, err
}

func (s *OptiScalerService) GetOptiScalerRecoveryState(_ context.Context) (result dto.OptiScalerRecoveryState, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get OptiScaler recovery state: %w", err)
		}
	}()
	return s.manager.RecoveryState()
}

func (s *OptiScalerService) RollbackOptiScalerRecovery(_ context.Context, journalID string) (result dto.OptiScalerApplyResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("rollback OptiScaler recovery state: %w", err)
		}
	}()
	return s.manager.RollbackRecovery(journalID)
}

func (s *OptiScalerService) requireWindows() error {
	if s.operatingSystem != "windows" {
		return errors.New("OptiScaler management is only supported on Windows")
	}
	return nil
}
