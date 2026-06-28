package services

import (
	"context"
	"fmt"
	"time"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/desired"
	"github.com/phergul/fiach/internal/deployment/drift"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/deployment/provenance"
	"github.com/phergul/fiach/internal/deployment/review"
	"github.com/phergul/fiach/internal/deployment/rules"
	"github.com/phergul/fiach/internal/operationplan"
	"github.com/phergul/fiach/internal/storage"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

type deploymentPlanBuildResult struct {
	Profile            dbtypes.ModProfile
	Resolved           operationplan.ResolveProfilePlanResult
	Desired            deployment.DesiredState
	Plan               planner.DeploymentPlan
	DeploymentRules    []rules.DeploymentRule
	PerFileWinnerRules map[string]rules.DeploymentRule
	AppliedAt          *time.Time
	AppliedFound       bool
}

func (s *DeploymentReviewService) buildDeploymentPlan(
	ctx context.Context,
	profileID int64,
) (result deploymentPlanBuildResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("build deployment plan: %w", err)
		}
	}()

	if profileID <= 0 {
		return deploymentPlanBuildResult{}, apperror.New("A valid profile must be selected.")
	}

	profile, found, err := s.store.GetProfile(ctx, profileID)
	if err != nil {
		return deploymentPlanBuildResult{}, err
	}
	if !found {
		return deploymentPlanBuildResult{}, apperror.New("Profile was not found.")
	}

	appliedState, appliedFound, err := s.store.GetAppliedProfileState(ctx, profile.GameID)
	if err != nil {
		return deploymentPlanBuildResult{}, err
	}
	if appliedFound && appliedState.ProfileID != profileID {
		return deploymentPlanBuildResult{}, apperror.New("Restore vanilla before applying another profile.")
	}

	resolved, err := operationplan.ResolveProfilePlan(ctx, s.store, profileID)
	if err != nil {
		return deploymentPlanBuildResult{}, err
	}

	deploymentRules, err := s.loadDeploymentRules(ctx, profileID)
	if err != nil {
		return deploymentPlanBuildResult{}, err
	}

	desiredState, err := desired.BuildDesiredState(ctx, resolved, deploymentRules)
	if err != nil {
		return deploymentPlanBuildResult{}, err
	}

	var plan planner.DeploymentPlan
	var appliedAt *time.Time

	if appliedFound && appliedState.ProfileID == profileID {
		appliedFileStates, err := s.profileService.LoadAppliedFileStates(ctx, profile.GameID)
		if err != nil {
			return deploymentPlanBuildResult{}, err
		}

		provenance.ReconcileModAddedPaths(&desiredState, appliedFileStates)

		driftResults, err := drift.DetectAll(resolved.GameInstallPath, appliedFileStates)
		if err != nil {
			return deploymentPlanBuildResult{}, err
		}

		plan, err = planner.PlanIncremental(desiredState, appliedFileStates, driftResults, resolved.GameInstallPath)
		if err != nil {
			return deploymentPlanBuildResult{}, err
		}

		if parsed, ok := storage.ParseAppliedTimestamp(appliedState.AppliedAt); ok {
			appliedAt = &parsed
		}
	} else {
		plan, err = planner.PlanFirstApply(desiredState, resolved.GameInstallPath)
		if err != nil {
			return deploymentPlanBuildResult{}, err
		}
	}

	return deploymentPlanBuildResult{
		Profile:            profile,
		Resolved:           resolved,
		Desired:            desiredState,
		Plan:               plan,
		DeploymentRules:    deploymentRules,
		PerFileWinnerRules: rules.IndexPerFileWinnerRules(deploymentRules),
		AppliedAt:          appliedAt,
		AppliedFound:       appliedFound && appliedState.ProfileID == profileID,
	}, nil
}

func (s *DeploymentReviewService) loadDeploymentRules(
	ctx context.Context,
	profileID int64,
) ([]rules.DeploymentRule, error) {
	rows, err := s.store.ListDeploymentRulesByProfileID(ctx, profileID)
	if err != nil {
		return nil, err
	}

	deploymentRules := make([]rules.DeploymentRule, 0, len(rows))
	for _, row := range rows {
		if row.WinnerModID == nil || *row.WinnerModID <= 0 {
			continue
		}
		deploymentRules = append(deploymentRules, rules.DeploymentRule{
			ProfileID:        row.ProfileID,
			GameRelativePath: row.GameRelativePath,
			RuleKind:         row.RuleKind,
			WinnerModID:      *row.WinnerModID,
		})
	}

	return deploymentRules, nil
}

func deploymentPlanPreviewEntry(result deploymentPlanBuildResult) (review.CachedPreview, error) {
	entry := review.CachedPreview{
		PreviewHash:        "",
		ProfileID:          result.Profile.ID,
		GameID:             result.Profile.GameID,
		ProfileName:        result.Profile.Name,
		Plan:               result.Plan,
		Desired:            result.Desired,
		PerFileWinnerRules: result.PerFileWinnerRules,
		AppliedAt:          result.AppliedAt,
	}

	previewHash, err := review.PreviewHash(entry)
	if err != nil {
		return review.CachedPreview{}, err
	}
	entry.PreviewHash = previewHash

	return entry, nil
}
