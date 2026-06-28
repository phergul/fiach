package services

import (
	"context"
	"log/slog"
	"strings"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/review"
	"github.com/phergul/fiach/internal/deployment/rules"
	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
)

func (s *DeploymentReviewService) SetDeploymentConflictRule(
	ctx context.Context,
	profileID int64,
	previewHash string,
	relativePath string,
	action string,
) (preview dto.DeploymentReviewPreview, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationSetDeploymentConflictRule, "Deployment conflict rule save started",
		slog.Int64("profile_id", profileID),
		slog.Bool("preview_hash_provided", strings.TrimSpace(previewHash) != ""),
		slog.String("relative_path", relativePath),
		slog.String("action", action),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Deployment conflict rule save failed", err, deploymentReviewUserError)
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

	desiredFile, hasDesired := buildResult.Desired.Files[canonicalPath]
	if !hasDesired {
		return dto.DeploymentReviewPreview{}, apperror.New("The selected file was not found in the deployment preview.")
	}

	var savedRule *rules.DeploymentRule
	if rule, ruleFound := buildResult.PerFileWinnerRules[canonicalPath]; ruleFound {
		ruleCopy := rule
		savedRule = &ruleCopy
	}

	conflictCategory := pathPlan.ConflictCategory
	if conflictCategory == "" {
		conflictCategory = desiredFile.ConflictCategory
	}

	availableActions := review.BuildAvailableConflictActions(desiredFile, conflictCategory, savedRule)
	if err := validateConflictActionInput(action, availableActions); err != nil {
		return dto.DeploymentReviewPreview{}, err
	}

	if rules.IsClearConflictRuleAction(action) {
		if err := s.store.DeletePerFileWinnerRule(ctx, profileID, pathPlan.GameRelativePath); err != nil {
			return dto.DeploymentReviewPreview{}, err
		}
	} else {
		winnerModID, ok := rules.ParseSetPerFileWinnerAction(action)
		if !ok {
			return dto.DeploymentReviewPreview{}, apperror.New("That conflict action is not allowed for this file.")
		}
		if err := s.store.UpsertPerFileWinnerRule(ctx, profileID, pathPlan.GameRelativePath, winnerModID); err != nil {
			return dto.DeploymentReviewPreview{}, err
		}
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

	diag.complete("Deployment conflict rule save completed",
		slog.String("preview_hash", preview.PreviewHash),
		slog.Bool("can_apply", preview.Summary.CanApply),
	)

	return preview, nil
}

func validateConflictActionInput(action string, availableActions []string) error {
	action = strings.TrimSpace(action)
	if action == "" {
		return apperror.New("A conflict action is required.")
	}
	if err := rules.ValidateConflictAction(action); err != nil {
		return apperror.New("That conflict action is not allowed for this file.")
	}

	for _, availableAction := range availableActions {
		if availableAction == action {
			return nil
		}
	}

	return apperror.New("That conflict action is not allowed for this file.")
}
