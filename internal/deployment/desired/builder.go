package desired

import (
	"context"
	"fmt"
	"sort"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/profile"
	"github.com/phergul/fiach/internal/deployment/provenance"
	"github.com/phergul/fiach/internal/deployment/rules"
)

type pathAccumulator struct {
	gameRelativePath string
	sourcePath       string
	sha256           string
	sizeBytes        int64
	writers          []deployment.WriterEntry
	contentByModID   map[int64]deployment.ModFileContent
}

func BuildDesiredState(
	ctx context.Context,
	resolved profile.ResolveProfilePlanResult,
	deploymentRules []rules.DeploymentRule,
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
		Issues:    append([]deployment.PlanIssue(nil), resolved.Issues...),
	}

	mods := append([]profile.ProfilePlanMod{}, resolved.Mods...)
	sort.SliceStable(mods, func(i int, j int) bool {
		if mods[i].LoadOrder != mods[j].LoadOrder {
			return mods[i].LoadOrder < mods[j].LoadOrder
		}
		return mods[i].ModID < mods[j].ModID
	})

	accumulators := map[string]*pathAccumulator{}

	for _, mod := range mods {
		inventory, inventoryErr := inventoryFilesForMod(profile.StrategyBuildInput{
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
					contentByModID: map[int64]deployment.ModFileContent{
						mod.ModID: {
							SourcePath: mapping.SourcePath,
							SHA256:     mapping.SHA256,
							SizeBytes:  mapping.SizeBytes,
						},
					},
				}
				continue
			}

			existing.sourcePath = mapping.SourcePath
			existing.sha256 = mapping.SHA256
			existing.sizeBytes = mapping.SizeBytes
			existing.gameRelativePath = mapping.GameRelativePath
			existing.writers = append(existing.writers, writer)
			if existing.contentByModID == nil {
				existing.contentByModID = map[int64]deployment.ModFileContent{}
			}
			existing.contentByModID[mod.ModID] = deployment.ModFileContent{
				SourcePath: mapping.SourcePath,
				SHA256:     mapping.SHA256,
				SizeBytes:  mapping.SizeBytes,
			}
		}
	}

	for key, accumulated := range accumulators {
		modContentByID := map[int64]deployment.ModFileContent{}
		for modID, content := range accumulated.contentByModID {
			modContentByID[modID] = content
		}

		state.Files[key] = deployment.DesiredFile{
			GameRelativePath: accumulated.gameRelativePath,
			SourcePath:       accumulated.sourcePath,
			SHA256:           accumulated.sha256,
			SizeBytes:        accumulated.sizeBytes,
			OutputKind:       deployment.OutputCopied,
			Writers:          append([]deployment.WriterEntry{}, accumulated.writers...),
			ModContentByID:   modContentByID,
		}
	}

	if err := provenance.EnrichState(&state, resolved.GameInstallPath, deploymentRules); err != nil {
		return deployment.DesiredState{}, err
	}

	return state, nil
}
