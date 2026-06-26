package desired

import (
	"context"
	"fmt"
	"sort"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/provenance"
	"github.com/phergul/fiach/internal/operationplan"
)

type pathAccumulator struct {
	gameRelativePath string
	sourcePath       string
	sha256           string
	sizeBytes        int64
	writers          []deployment.WriterEntry
}

func BuildDesiredState(
	ctx context.Context,
	resolved operationplan.ResolveProfilePlanResult,
) (state deployment.DesiredState, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("build desired state: %w", err)
		}
	}()

	_ = ctx

	state = deployment.DesiredState{
		ProfileID: resolved.ProfileID,
		GameID:    resolved.GameID,
		Files:     map[string]deployment.DesiredFile{},
		Issues:    mapResolvedIssues(resolved),
	}

	mods := append([]operationplan.ProfilePlanMod{}, resolved.Mods...)
	sort.SliceStable(mods, func(i int, j int) bool {
		if mods[i].LoadOrder != mods[j].LoadOrder {
			return mods[i].LoadOrder < mods[j].LoadOrder
		}
		return mods[i].ModID < mods[j].ModID
	})

	accumulators := map[string]*pathAccumulator{}

	for _, mod := range mods {
		inventory, inventoryErr := inventoryFilesForMod(operationplan.StrategyBuildInput{
			ProfileID:          resolved.ProfileID,
			GameInstallPath:    resolved.GameInstallPath,
			GameModStoragePath: resolved.GameModStoragePath,
			Mod:                mod,
		})
		if inventoryErr != nil {
			return deployment.DesiredState{}, inventoryErr
		}

		state.Issues = append(state.Issues, inventory.Issues...)

		for _, mapping := range inventory.Mappings {
			key := deployment.CanonicalGameRelativePath(mapping.GameRelativePath)
			writer := provenance.NewModWriter(mod.ModID, mod.ModName, mod.LoadOrder)

			existing, found := accumulators[key]
			if !found {
				accumulators[key] = &pathAccumulator{
					gameRelativePath: mapping.GameRelativePath,
					sourcePath:       mapping.SourcePath,
					sha256:           mapping.SHA256,
					sizeBytes:        mapping.SizeBytes,
					writers:          []deployment.WriterEntry{writer},
				}
				continue
			}

			existing.sourcePath = mapping.SourcePath
			existing.sha256 = mapping.SHA256
			existing.sizeBytes = mapping.SizeBytes
			existing.gameRelativePath = mapping.GameRelativePath
			existing.writers = append(existing.writers, writer)
		}
	}

	for key, accumulated := range accumulators {
		state.Files[key] = deployment.DesiredFile{
			GameRelativePath: accumulated.gameRelativePath,
			SourcePath:       accumulated.sourcePath,
			SHA256:           accumulated.sha256,
			SizeBytes:        accumulated.sizeBytes,
			OutputKind:       deployment.OutputCopied,
			Writers:          append([]deployment.WriterEntry{}, accumulated.writers...),
		}
	}

	if err := provenance.EnrichState(&state, resolved.GameInstallPath); err != nil {
		return deployment.DesiredState{}, err
	}

	return state, nil
}

func mapResolvedIssues(resolved operationplan.ResolveProfilePlanResult) []deployment.PlanIssue {
	if len(resolved.Issues) == 0 {
		return nil
	}

	issues := make([]deployment.PlanIssue, 0, len(resolved.Issues))
	for _, issue := range resolved.Issues {
		mapped, ok := mapOperationPlanIssue(issue)
		if !ok {
			continue
		}
		issues = append(issues, mapped)
	}
	return issues
}

func mapOperationPlanIssue(issue operationplan.PlanIssue) (deployment.PlanIssue, bool) {
	kind, ok := mapOperationPlanIssueKind(issue.Kind)
	if !ok {
		return deployment.PlanIssue{}, false
	}

	mapped := deployment.PlanIssue{
		Severity:  deployment.PlanIssueSeverity(issue.Severity),
		Kind:      kind,
		Message:   issue.Message,
		ProfileID: issue.ProfileID,
	}
	if issue.SourcePath != nil {
		value := *issue.SourcePath
		mapped.SourcePath = &value
	}
	if issue.TargetPath != nil {
		value := *issue.TargetPath
		mapped.TargetPath = &value
	}
	if issue.Mod != nil {
		mapped.Mod = &deployment.ModContext{
			ModID:   issue.Mod.ModID,
			ModName: issue.Mod.ModName,
		}
	}
	return mapped, true
}

func mapOperationPlanIssueKind(kind operationplan.PlanIssueKind) (deployment.PlanIssueKind, bool) {
	switch kind {
	case operationplan.PlanIssueMissingManagedSourcePath:
		return deployment.PlanIssueMissingManagedSourcePath, true
	case operationplan.PlanIssueMissingInstallConfig:
		return deployment.PlanIssueMissingInstallConfig, true
	case operationplan.PlanIssueIncompleteInstallConfig:
		return deployment.PlanIssueIncompleteInstallConfig, true
	default:
		return "", false
	}
}
