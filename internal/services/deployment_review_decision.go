package services

import (
	"context"
	"log/slog"
	"strings"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/drift"
	"github.com/phergul/fiach/internal/deployment/review"
	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
)

func (s *DeploymentReviewService) SetDeploymentDriftDecision(
	ctx context.Context,
	profileID int64,
	previewHash string,
	relativePath string,
	decision string,
) (preview dto.DeploymentReviewPreview, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationSetDeploymentDriftDecision, "Deployment drift decision save started",
		slog.Int64("profile_id", profileID),
		slog.Bool("preview_hash_provided", strings.TrimSpace(previewHash) != ""),
		slog.String("relative_path", relativePath),
		slog.String("decision", decision),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Deployment drift decision save failed", err, deploymentReviewUserError)
		}
	}()

	if strings.TrimSpace(previewHash) == "" {
		return dto.DeploymentReviewPreview{}, apperror.New("Refresh the deployment preview and try again.")
	}

	cachedPreview, err := s.requireCachedPreview(previewHash)
	if err != nil {
		return dto.DeploymentReviewPreview{}, err
	}
	if cachedPreview.ProfileID != profileID {
		return dto.DeploymentReviewPreview{}, apperror.New("The deployment preview is stale. Refresh the preview and try again.")
	}

	buildResult, err := s.buildDeploymentPlan(ctx, profileID)
	if err != nil {
		return dto.DeploymentReviewPreview{}, err
	}

	freshEntry, err := deploymentPlanPreviewEntry(buildResult)
	if err != nil {
		return dto.DeploymentReviewPreview{}, err
	}
	if !strings.EqualFold(freshEntry.PreviewHash, previewHash) {
		return dto.DeploymentReviewPreview{}, apperror.New("The deployment preview is stale. Refresh the preview and try again.")
	}

	canonicalPath := deployment.CanonicalGameRelativePath(relativePath)
	pathPlan, found := buildResult.Plan.Paths[canonicalPath]
	if !found {
		return dto.DeploymentReviewPreview{}, apperror.New("The selected file was not found in the deployment preview.")
	}

	_, hasDesired := buildResult.Desired.Files[canonicalPath]
	availableActions := review.BuildAvailableDriftActions(pathPlan, hasDesired)
	if err := validateDriftDecisionInput(decision, availableActions); err != nil {
		return dto.DeploymentReviewPreview{}, err
	}

	storedDecision, err := decisionToStoredValue(decision)
	if err != nil {
		return dto.DeploymentReviewPreview{}, err
	}

	if err := s.store.UpdateAppliedFileStateUserDecision(
		ctx,
		buildResult.Profile.GameID,
		profileID,
		pathPlan.GameRelativePath,
		storedDecision,
	); err != nil {
		return dto.DeploymentReviewPreview{}, err
	}

	s.cache.Delete(previewHash)

	rebuiltResult, err := s.buildDeploymentPlan(ctx, profileID)
	if err != nil {
		return dto.DeploymentReviewPreview{}, err
	}

	entry, err := deploymentPlanPreviewEntry(rebuiltResult)
	if err != nil {
		return dto.DeploymentReviewPreview{}, err
	}

	s.cache.Store(entry)

	rootNodes := review.BuildTreeChildren(rebuiltResult.Plan, "")
	preview = mappers.ToDTODeploymentReviewPreview(entry, rootNodes)

	diag.complete("Deployment drift decision save completed",
		slog.String("preview_hash", preview.PreviewHash),
		slog.Bool("can_apply", preview.Summary.CanApply),
	)

	return preview, nil
}

func validateDriftDecisionInput(decision string, availableActions []string) error {
	decision = strings.TrimSpace(decision)
	if decision == "" {
		return apperror.New("A drift decision is required.")
	}

	for _, availableAction := range availableActions {
		if availableAction == decision {
			return nil
		}
	}

	return apperror.New("That drift decision is not allowed for this file.")
}

func decisionToStoredValue(decision string) (*string, error) {
	decision = strings.TrimSpace(decision)
	if drift.IsClearInput(decision) {
		return nil, nil
	}
	if !drift.IsPersistedDecision(decision) {
		return nil, apperror.New("That drift decision is not allowed for this file.")
	}

	return &decision, nil
}
