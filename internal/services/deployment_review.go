package services

import (
	"context"
	"log/slog"
	"strings"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/deployment/desired"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/deployment/review"
	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/operationplan"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
	"github.com/phergul/fiach/internal/storage"
)

type DeploymentReviewService struct {
	store  *storage.Store
	logger *slog.Logger
	cache  *review.PreviewCache
}

func NewDeploymentReviewService(store *storage.Store, logger *slog.Logger) *DeploymentReviewService {
	if logger == nil {
		logger = slog.Default()
	}

	return &DeploymentReviewService{
		store:  store,
		logger: logger,
		cache:  review.NewPreviewCache(),
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

	if profileID <= 0 {
		return dto.DeploymentReviewPreview{}, apperror.New("A valid profile must be selected.")
	}

	profile, found, err := s.store.GetProfile(ctx, profileID)
	if err != nil {
		return dto.DeploymentReviewPreview{}, err
	}
	if !found {
		return dto.DeploymentReviewPreview{}, apperror.New("Profile was not found.")
	}

	if _, appliedFound, err := s.store.GetAppliedProfileState(ctx, profile.GameID); err != nil {
		return dto.DeploymentReviewPreview{}, err
	} else if appliedFound {
		return dto.DeploymentReviewPreview{}, apperror.New("Restore vanilla before applying another profile.")
	}

	resolved, err := operationplan.ResolveProfilePlan(ctx, s.store, profileID)
	if err != nil {
		return dto.DeploymentReviewPreview{}, err
	}

	desiredState, err := desired.BuildDesiredState(ctx, resolved)
	if err != nil {
		return dto.DeploymentReviewPreview{}, err
	}

	firstApplyPlan, err := planner.PlanFirstApply(desiredState, resolved.GameInstallPath)
	if err != nil {
		return dto.DeploymentReviewPreview{}, err
	}

	entry := review.CachedPreview{
		ProfileID:   profileID,
		GameID:      profile.GameID,
		ProfileName: profile.Name,
		Plan:        firstApplyPlan,
		Desired:     desiredState,
	}

	previewHash, err := review.PreviewHash(entry)
	if err != nil {
		return dto.DeploymentReviewPreview{}, err
	}
	entry.PreviewHash = previewHash

	s.cache.Store(entry)

	rootNodes := review.BuildTreeChildren(firstApplyPlan, "")
	preview = mappers.ToDTODeploymentReviewPreview(entry, rootNodes)

	diag.complete("Deployment review preview build completed",
		slog.Bool("can_apply", preview.Summary.CanApply),
		slog.Int("path_count", len(firstApplyPlan.Paths)),
		slog.Int("blocking_count", preview.Summary.BlockingCount),
	)

	return preview, nil
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

	detail = mappers.ToDTODeploymentFileDetail(fileDetail)

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
	case strings.Contains(message, "restore vanilla before applying another profile"):
		return apperror.Wrap("Restore vanilla before applying another profile.", err)
	case strings.Contains(message, "was not found in preview"):
		return apperror.Wrap("The selected file was not found in the deployment preview.", err)
	default:
		return err
	}
}
