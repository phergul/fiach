package operationplan

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/phergul/fiach/internal/installconfig"
	"github.com/phergul/fiach/internal/storage"
)

type ProfilePlanMod struct {
	ProfileID          int64
	ModID              int64
	ModName            string
	ManagedSourcePath  string
	LoadOrder          int64
	StrategyType       installconfig.StrategyType
	TargetBase         string
	TargetRelativePath string
	SourceSubpath      *string
}

type ResolveProfilePlanResult struct {
	ProfileID          int64
	GameID             int64
	GameInstallPath    string
	GameModStoragePath string
	Mods               []ProfilePlanMod
	Issues             []PlanIssue
}

func ResolveProfilePlan(ctx context.Context, store *storage.Store, profileID int64) (result ResolveProfilePlanResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("resolve profile plan: %w", err)
		}
	}()

	if store == nil {
		return ResolveProfilePlanResult{}, errors.New("store is not configured")
	}

	profile, found, err := store.GetProfile(ctx, profileID)
	if err != nil {
		return ResolveProfilePlanResult{}, err
	}
	if !found {
		return ResolveProfilePlanResult{}, fmt.Errorf("profile %d was not found", profileID)
	}

	game, err := store.GetStoredGame(ctx, profile.GameID)
	if err != nil {
		return ResolveProfilePlanResult{}, err
	}

	gameModStoragePath, err := store.ResolveGameModStoragePath(ctx, profile.GameID, "")
	if err != nil {
		return ResolveProfilePlanResult{}, err
	}

	result.ProfileID = profileID
	result.GameID = profile.GameID
	result.GameInstallPath = game.InstallPath
	result.GameModStoragePath = gameModStoragePath

	profileMods, err := store.ListProfileMods(ctx, profileID)
	if err != nil {
		return ResolveProfilePlanResult{}, err
	}

	for _, profileMod := range profileMods {
		if !profileMod.Enabled {
			continue
		}

		managedSourcePath := strings.TrimSpace(profileMod.SourcePath)
		if managedSourcePath == "" {
			result.Issues = append(result.Issues, newPlanIssue(
				PlanIssueSeverityError,
				PlanIssueMissingManagedSourcePath,
				profileID,
				fmt.Sprintf("mod %q is missing a managed source path", profileMod.Name),
				modContextPtr(profileMod.ModID, profileMod.Name),
				nil,
				nil,
			))
			continue
		}

		config, found, err := store.GetModInstallConfig(ctx, profileMod.ModID)
		if err != nil {
			return ResolveProfilePlanResult{}, err
		}
		if !found {
			result.Issues = append(result.Issues, newPlanIssue(
				PlanIssueSeverityError,
				PlanIssueMissingInstallConfig,
				profileID,
				fmt.Sprintf("mod %q is missing an install configuration", profileMod.Name),
				modContextPtr(profileMod.ModID, profileMod.Name),
				nil,
				nil,
			))
			continue
		}

		if strings.TrimSpace(config.StrategyType) == "" || strings.TrimSpace(config.TargetBase) == "" || strings.TrimSpace(config.TargetRelativePath) == "" {
			result.Issues = append(result.Issues, newPlanIssue(
				PlanIssueSeverityError,
				PlanIssueIncompleteInstallConfig,
				profileID,
				fmt.Sprintf("mod %q has an incomplete install configuration", profileMod.Name),
				modContextPtr(profileMod.ModID, profileMod.Name),
				nil,
				nil,
			))
			continue
		}

		result.Mods = append(result.Mods, ProfilePlanMod{
			ProfileID:          profileID,
			ModID:              profileMod.ModID,
			ModName:            profileMod.Name,
			ManagedSourcePath:  managedSourcePath,
			LoadOrder:          profileMod.LoadOrder,
			StrategyType:       installconfig.StrategyType(config.StrategyType),
			TargetBase:         config.TargetBase,
			TargetRelativePath: config.TargetRelativePath,
			SourceSubpath:      config.SourceSubpath,
		})
	}

	return result, nil
}

func newPlanIssue(
	severity PlanIssueSeverity,
	kind PlanIssueKind,
	profileID int64,
	message string,
	mod *ModContext,
	sourcePath *string,
	targetPath *string,
) PlanIssue {
	return PlanIssue{
		Severity:   severity,
		Kind:       kind,
		Message:    message,
		ProfileID:  profileID,
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Mod:        mod,
	}
}

func modContextPtr(modID int64, modName string) *ModContext {
	return &ModContext{
		ModID:   modID,
		ModName: modName,
	}
}

func stringPtr(value string) *string {
	return &value
}
