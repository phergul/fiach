package services

import (
	"context"
	"fmt"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/execute"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func (s *ProfileService) SaveIncrementalAppliedProfileState(
	ctx context.Context,
	gameID int64,
	profileID int64,
	installPath string,
	plan planner.DeploymentPlan,
	desired deployment.DesiredState,
	existingStates []appliedstate.PersistedFileState,
) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("save incremental applied profile state: %w", err)
		}
	}()

	appliedState, found, err := s.store.GetAppliedProfileState(ctx, gameID)
	if err != nil {
		return err
	}
	if !found || appliedState.ProfileID != profileID {
		return fmt.Errorf("applied profile state for game %d profile %d was not found", gameID, profileID)
	}

	mergedStates, err := execute.MergeAppliedFileStates(plan, desired, existingStates, profileID)
	if err != nil {
		return err
	}

	manifestDocument := appliedstate.ManifestDocument{
		Version: appliedstate.DocumentVersion,
	}
	appliedstate.AttachManifestFiles(&manifestDocument, mergedStates)

	manifestJSON, err := appliedstate.EncodeManifest(manifestDocument)
	if err != nil {
		return fmt.Errorf("encode applied manifest: %w", err)
	}

	profileMods, err := s.store.ListProfileMods(ctx, profileID)
	if err != nil {
		return err
	}
	compositionSnapshot, err := encodeProfileCompositionSnapshot(profileID, profileMods)
	if err != nil {
		return fmt.Errorf("encode profile composition snapshot: %w", err)
	}

	_, err = s.store.SaveAppliedProfileState(ctx, dbtypes.SaveAppliedProfileStateInput{
		GameID:                         gameID,
		ProfileID:                      profileID,
		ManifestJSON:                   manifestJSON,
		ProfileSnapshotJSON:            appliedState.ProfileSnapshotJSON,
		ProfileSnapshotHash:            appliedState.ProfileSnapshotHash,
		ProfileCompositionSnapshotJSON: &compositionSnapshot.JSON,
		ProfileCompositionSnapshotHash: &compositionSnapshot.Hash,
		FileStates:                     toDBAppliedFileStateRows(gameID, mergedStates, ""),
		ReplaceFileStates:              true,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *ProfileService) SaveFirstApplyAppliedProfileState(
	ctx context.Context,
	gameID int64,
	profileID int64,
	installPath string,
	plan planner.DeploymentPlan,
	desired deployment.DesiredState,
	outcome execute.FirstApplyOutcome,
	previewHash string,
) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("save first apply applied profile state: %w", err)
		}
	}()

	if _, found, err := s.store.GetAppliedProfileState(ctx, gameID); err != nil {
		return err
	} else if found {
		return apperror.New("Restore vanilla before applying another profile.")
	}

	fileStates, err := execute.BuildInitialAppliedFileStates(plan, desired, outcome, profileID)
	if err != nil {
		return err
	}

	manifestDocument, err := execute.BuildFirstApplyManifest(plan, desired, outcome, installPath)
	if err != nil {
		return err
	}
	appliedstate.AttachManifestFiles(&manifestDocument, fileStates)

	manifestJSON, err := appliedstate.EncodeManifest(manifestDocument)
	if err != nil {
		return fmt.Errorf("encode applied manifest: %w", err)
	}

	snapshot, err := appliedstate.EncodeDeploymentProfileSnapshot(
		appliedstate.BuildDeploymentProfileSnapshotDocument(previewHash, string(plan.Mode)),
	)
	if err != nil {
		return fmt.Errorf("encode profile snapshot: %w", err)
	}

	profileMods, err := s.store.ListProfileMods(ctx, profileID)
	if err != nil {
		return err
	}
	compositionSnapshot, err := encodeProfileCompositionSnapshot(profileID, profileMods)
	if err != nil {
		return fmt.Errorf("encode profile composition snapshot: %w", err)
	}

	_, err = s.store.SaveAppliedProfileState(ctx, dbtypes.SaveAppliedProfileStateInput{
		GameID:                         gameID,
		ProfileID:                      profileID,
		ManifestJSON:                   manifestJSON,
		ProfileSnapshotJSON:            snapshot.JSON,
		ProfileSnapshotHash:            snapshot.Hash,
		ProfileCompositionSnapshotJSON: &compositionSnapshot.JSON,
		ProfileCompositionSnapshotHash: &compositionSnapshot.Hash,
		FileStates:                     toDBAppliedFileStateRows(gameID, fileStates, ""),
		ReplaceFileStates:              true,
	})
	if err != nil {
		return err
	}

	return nil
}

var _ execute.AppliedStateSaver = (*ProfileService)(nil)
