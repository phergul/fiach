package services

import (
	"context"
	"log/slog"
	"strings"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/deployment/execute"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/deployment/review"
	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
	"github.com/phergul/fiach/internal/storage"
)

type DeploymentReviewService struct {
	store          *storage.Store
	profileService *ProfileService
	logger         *slog.Logger
	cache          *review.PreviewCache
}

func NewDeploymentReviewService(store *storage.Store, profileService *ProfileService, logger *slog.Logger) *DeploymentReviewService {
	if logger == nil {
		logger = slog.Default()
	}

	return &DeploymentReviewService{
		store:          store,
		profileService: profileService,
		logger:         logger,
		cache:          review.NewPreviewCache(),
	}
}

func (s *DeploymentReviewService) BuildDeploymentReviewPreview(ctx context.Context, profileID int64) (preview dto.DeploymentReviewPreview, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationBuildDeploymentReviewPreview, "Deployment review preview build started",
		slog.Int64("profile_id", profileID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Deployment review preview build failed", err, deploymentReviewUserError)
		}
	}()

	buildResult, err := s.buildDeploymentPlan(ctx, profileID)
	if err != nil {
		return dto.DeploymentReviewPreview{}, err
	}

	entry, err := deploymentPlanPreviewEntry(buildResult)
	if err != nil {
		return dto.DeploymentReviewPreview{}, err
	}

	s.cache.Store(entry)

	rootNodes := review.BuildTreeChildren(buildResult.Plan, "")
	preview = mappers.ToDTODeploymentReviewPreview(entry, rootNodes)

	diag.complete("Deployment review preview build completed",
		slog.Bool("can_apply", preview.Summary.CanApply),
		slog.Int("path_count", len(buildResult.Plan.Paths)),
		slog.Int("blocking_count", preview.Summary.BlockingCount),
	)

	return preview, nil
}

func (s *DeploymentReviewService) ApplyIncrementalDeployment(
	ctx context.Context,
	profileID int64,
	previewHash string,
) (result dto.ApplyIncrementalDeploymentResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationApplyIncrementalDeployment, "Incremental deployment apply started",
		slog.Int64("profile_id", profileID),
		slog.Bool("preview_hash_provided", strings.TrimSpace(previewHash) != ""),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Incremental deployment apply failed", err, deploymentReviewUserError)
		}
	}()

	if strings.TrimSpace(previewHash) == "" {
		return dto.ApplyIncrementalDeploymentResult{}, apperror.New("Refresh the deployment preview and try again.")
	}

	buildResult, err := s.buildDeploymentPlan(ctx, profileID)
	if err != nil {
		return dto.ApplyIncrementalDeploymentResult{}, err
	}

	if !buildResult.AppliedFound {
		return dto.ApplyIncrementalDeploymentResult{}, apperror.New("No profile is currently applied for this game.")
	}
	if buildResult.Plan.Mode != planner.PlanModeIncremental {
		return dto.ApplyIncrementalDeploymentResult{}, apperror.New("Incremental deployment apply is only available for an already applied profile.")
	}

	entry, err := deploymentPlanPreviewEntry(buildResult)
	if err != nil {
		return dto.ApplyIncrementalDeploymentResult{}, err
	}
	if !strings.EqualFold(entry.PreviewHash, previewHash) {
		return dto.ApplyIncrementalDeploymentResult{}, apperror.New("The deployment preview is stale. Refresh the preview and try again.")
	}
	if !buildResult.Plan.CanApply() {
		return dto.ApplyIncrementalDeploymentResult{}, apperror.New("Resolve blocking issues before applying this profile.")
	}

	gameModStoragePath, err := s.store.ResolveGameModStoragePath(ctx, buildResult.Profile.GameID, "")
	if err != nil {
		return dto.ApplyIncrementalDeploymentResult{}, err
	}

	appliedFileStates, err := s.profileService.LoadAppliedFileStates(ctx, buildResult.Profile.GameID)
	if err != nil {
		return dto.ApplyIncrementalDeploymentResult{}, err
	}

	applyResult, err := execute.Execute(ctx, execute.Context{
		GameID:             buildResult.Profile.GameID,
		ProfileID:          profileID,
		GameInstallPath:    buildResult.Resolved.GameInstallPath,
		GameModStoragePath: gameModStoragePath,
		PreviewHash:        previewHash,
		Plan:               buildResult.Plan,
		Desired:            buildResult.Desired,
		AppliedFileStates:  appliedFileStates,
	}, s.profileService)
	if err != nil {
		result = mappers.ToDTOApplyIncrementalDeploymentResult(applyResult)
		diag.warn("Incremental deployment apply completed with failures",
			slog.Bool("success", false),
			slog.Int("completed_count", applyResult.CompletedCount),
			slog.Bool("rolled_back", applyResult.RolledBack),
		)
		return result, err
	}

	s.cache.Delete(previewHash)

	result = mappers.ToDTOApplyIncrementalDeploymentResult(applyResult)
	diag.complete("Incremental deployment apply completed",
		slog.Bool("success", true),
		slog.Int("completed_count", applyResult.CompletedCount),
		slog.Int("skipped_count", applyResult.SkippedCount),
	)

	return result, nil
}

func (s *DeploymentReviewService) LoadDeploymentTreeChildren(ctx context.Context, previewHash string, parentPath string) (nodes []dto.DeploymentTreeNode, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationLoadDeploymentTreeChildren, "Deployment review tree children load started",
		slog.Bool("preview_hash_provided", strings.TrimSpace(previewHash) != ""),
		slog.String("parent_path", parentPath),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Deployment review tree children load failed", err, deploymentReviewUserError)
		}
	}()

	entry, err := s.requireCachedPreview(previewHash)
	if err != nil {
		return nil, err
	}

	treeNodes := review.BuildTreeChildren(entry.Plan, parentPath)
	nodes = mappers.ToDTODeploymentTreeNodes(treeNodes)

	diag.complete("Deployment review tree children load completed",
		slog.Int("child_count", len(nodes)),
	)

	return nodes, nil
}

func (s *DeploymentReviewService) GetDeploymentFileDetail(ctx context.Context, previewHash string, relativePath string) (detail dto.DeploymentFileDetail, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationGetDeploymentFileDetail, "Deployment review file detail load started",
		slog.Bool("preview_hash_provided", strings.TrimSpace(previewHash) != ""),
		slog.String("relative_path", relativePath),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Deployment review file detail load failed", err, deploymentReviewUserError)
		}
	}()

	entry, err := s.requireCachedPreview(previewHash)
	if err != nil {
		return dto.DeploymentFileDetail{}, err
	}

	fileDetail, err := review.BuildFileDetail(entry, relativePath)
	if err != nil {
		return dto.DeploymentFileDetail{}, err
	}

	detail = mappers.ToDTODeploymentFileDetail(fileDetail, entry.GameID)

	diag.complete("Deployment review file detail load completed",
		slog.String("relative_path", detail.RelativePath),
	)

	return detail, nil
}

func (s *DeploymentReviewService) requireCachedPreview(previewHash string) (review.CachedPreview, error) {
	if strings.TrimSpace(previewHash) == "" {
		return review.CachedPreview{}, apperror.New("Refresh the deployment preview and try again.")
	}

	entry, found := s.cache.Get(previewHash)
	if !found {
		return review.CachedPreview{}, apperror.New("The deployment preview is no longer available. Refresh the preview and try again.")
	}

	return entry, nil
}

func deploymentReviewUserError(err error) error {
	if err == nil {
		return nil
	}
	if apperror.UserMessage(err) != "" {
		return err
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "profile ID must be positive"):
		return apperror.Wrap("A valid profile must be selected.", err)
	case strings.Contains(message, "was not found") && strings.Contains(message, "profile"):
		return apperror.Wrap("Profile was not found.", err)
	case strings.Contains(strings.ToLower(message), "restore vanilla before applying another profile"):
		return apperror.Wrap("Restore vanilla before applying another profile.", err)
	case strings.Contains(message, "was not found in preview"):
		return apperror.Wrap("The selected file was not found in the deployment preview.", err)
	case strings.Contains(message, "preview is stale"):
		return apperror.Wrap("The deployment preview is stale. Refresh the preview and try again.", err)
	default:
		return err
	}
}
