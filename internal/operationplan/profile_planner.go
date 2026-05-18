package operationplan

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/phergul/mod-manager/internal/installconfig"
	"github.com/phergul/mod-manager/internal/storage"
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

type ProfilePlanIssueKind string

const (
	ProfilePlanIssueMissingManagedSourcePath ProfilePlanIssueKind = "missing_managed_source_path"
	ProfilePlanIssueMissingInstallConfig     ProfilePlanIssueKind = "missing_install_config"
	ProfilePlanIssueIncompleteInstallConfig  ProfilePlanIssueKind = "incomplete_install_config"
)

type ProfilePlanIssue struct {
	Kind      ProfilePlanIssueKind
	ProfileID int64
	ModID     int64
	ModName   string
	Message   string
}

type ResolveProfilePlanResult struct {
	Mods   []ProfilePlanMod
	Issues []ProfilePlanIssue
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

	_, found, err := store.GetProfile(ctx, profileID)
	if err != nil {
		return ResolveProfilePlanResult{}, err
	}
	if !found {
		return ResolveProfilePlanResult{}, fmt.Errorf("profile %d was not found", profileID)
	}

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
			result.Issues = append(result.Issues, ProfilePlanIssue{
				Kind:      ProfilePlanIssueMissingManagedSourcePath,
				ProfileID: profileID,
				ModID:     profileMod.ModID,
				ModName:   profileMod.Name,
				Message:   fmt.Sprintf("mod %q is missing a managed source path", profileMod.Name),
			})
			continue
		}

		config, found, err := store.GetModInstallConfig(ctx, profileMod.ModID)
		if err != nil {
			return ResolveProfilePlanResult{}, err
		}
		if !found {
			result.Issues = append(result.Issues, ProfilePlanIssue{
				Kind:      ProfilePlanIssueMissingInstallConfig,
				ProfileID: profileID,
				ModID:     profileMod.ModID,
				ModName:   profileMod.Name,
				Message:   fmt.Sprintf("mod %q is missing an install configuration", profileMod.Name),
			})
			continue
		}

		if strings.TrimSpace(config.StrategyType) == "" || strings.TrimSpace(config.TargetBase) == "" || strings.TrimSpace(config.TargetRelativePath) == "" {
			result.Issues = append(result.Issues, ProfilePlanIssue{
				Kind:      ProfilePlanIssueIncompleteInstallConfig,
				ProfileID: profileID,
				ModID:     profileMod.ModID,
				ModName:   profileMod.Name,
				Message:   fmt.Sprintf("mod %q has an incomplete install configuration", profileMod.Name),
			})
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
