package services

import (
	"context"
	"testing"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/deployment/rules"
	"github.com/phergul/fiach/internal/installconfig"
	"github.com/phergul/fiach/internal/services/dto"
)

func TestDeploymentReviewServiceSetConflictRuleResolvesAmbiguousOverwrite(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")

	firstSource := makeProfilePlanSourceTree(t, map[string]string{"plugin.txt": "alpha"})
	secondSource := makeProfilePlanSourceTree(t, map[string]string{"plugin.txt": "beta"})

	firstModID := insertServiceProfileTestMod(t, store, gameID, "Alpha", firstSource)
	secondModID := insertServiceProfileTestMod(t, store, gameID, "Beta", secondSource)

	addServiceProfileMod(t, store, profileID, firstModID, true, 0)
	addServiceProfileMod(t, store, profileID, secondModID, true, 0)
	addServiceInstallConfig(t, store, firstModID, string(dto.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Shared", nil)
	addServiceInstallConfig(t, store, secondModID, string(dto.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Shared", nil)

	service := newDeploymentReviewTestService(store)
	preview, err := service.BuildDeploymentReviewPreview(context.Background(), profileID)
	if err != nil {
		t.Fatalf("BuildDeploymentReviewPreview() error = %v", err)
	}
	if preview.Summary.CanApply {
		t.Fatal("initial preview CanApply = true, want false for ambiguous overwrite")
	}

	detail, err := service.GetDeploymentFileDetail(context.Background(), preview.PreviewHash, "Shared/plugin.txt")
	if err != nil {
		t.Fatalf("GetDeploymentFileDetail() error = %v", err)
	}
	if len(detail.ConflictAvailableActions) < 2 {
		t.Fatalf("ConflictAvailableActions = %+v, want per-mod actions", detail.ConflictAvailableActions)
	}

	updatedPreview, err := service.SetDeploymentConflictRule(
		context.Background(),
		profileID,
		preview.PreviewHash,
		"Shared/plugin.txt",
		rules.FormatSetPerFileWinnerAction(firstModID),
	)
	if err != nil {
		t.Fatalf("SetDeploymentConflictRule() error = %v", err)
	}
	if !updatedPreview.Summary.CanApply {
		t.Fatal("updated preview CanApply = false, want true after per-file winner rule")
	}

	updatedDetail, err := service.GetDeploymentFileDetail(
		context.Background(),
		updatedPreview.PreviewHash,
		"Shared/plugin.txt",
	)
	if err != nil {
		t.Fatalf("GetDeploymentFileDetail(updated) error = %v", err)
	}
	if updatedDetail.SavedConflictRuleModID == nil || *updatedDetail.SavedConflictRuleModID != firstModID {
		t.Fatalf("SavedConflictRuleModID = %+v, want %d", updatedDetail.SavedConflictRuleModID, firstModID)
	}
	if updatedDetail.SavedConflictRuleModName != "Alpha" {
		t.Fatalf("SavedConflictRuleModName = %q, want Alpha", updatedDetail.SavedConflictRuleModName)
	}
	if updatedDetail.ProfileModsURL == "" {
		t.Fatal("ProfileModsURL is empty, want profile link")
	}

	clearedPreview, err := service.SetDeploymentConflictRule(
		context.Background(),
		profileID,
		updatedPreview.PreviewHash,
		"Shared/plugin.txt",
		rules.ActionClearConflictRule,
	)
	if err != nil {
		t.Fatalf("SetDeploymentConflictRule(clear) error = %v", err)
	}
	if clearedPreview.Summary.CanApply {
		t.Fatal("cleared preview CanApply = true, want false after clearing rule")
	}
}

func TestDeploymentReviewServiceSetConflictRuleRejectsStalePreview(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", t.TempDir())
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	service := newDeploymentReviewTestService(store)

	_, err := service.SetDeploymentConflictRule(
		context.Background(),
		profileID,
		"stale-hash",
		"Shared/plugin.txt",
		rules.ActionClearConflictRule,
	)
	if err == nil {
		t.Fatal("SetDeploymentConflictRule() error = nil, want stale preview rejection")
	}
	if apperror.UserMessage(err) != "The deployment preview is no longer available. Refresh the preview and try again." {
		t.Fatalf("SetDeploymentConflictRule() error = %q, want stale preview rejection", apperror.UserMessage(err))
	}
}
