package services

import (
	"context"
	"fmt"

	"github.com/phergul/mod-manager/internal/appliedstate"
	"github.com/phergul/mod-manager/internal/storage"
)

type ProfileService struct {
	store *storage.Store
}

type AppliedProfileSummary struct {
	GameID                   int64
	ProfileID                int64
	ProfileName              string
	AppliedAt                string
	HasAppliedProfileChanged *bool
}

func NewProfileService(store *storage.Store) *ProfileService {
	return &ProfileService{
		store: store,
	}
}

func (s *ProfileService) CreateProfile(ctx context.Context, gameID int64, name string) (profile storage.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("create profile: %w", err)
		}
	}()

	return s.store.CreateProfile(ctx, gameID, name)
}

func (s *ProfileService) ListProfiles(ctx context.Context, gameID int64) (profiles []storage.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list profiles: %w", err)
		}
	}()

	return s.store.ListProfiles(ctx, gameID)
}

func (s *ProfileService) RenameProfile(ctx context.Context, profileID int64, name string) (profile storage.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("rename profile: %w", err)
		}
	}()

	return s.store.RenameProfile(ctx, profileID, name)
}

func (s *ProfileService) DeleteProfile(ctx context.Context, profileID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("delete profile: %w", err)
		}
	}()

	profile, found, err := s.store.GetProfile(ctx, profileID)
	if err != nil {
		return err
	}
	if found {
		appliedState, appliedFound, err := s.store.GetAppliedProfileState(ctx, profile.GameID)
		if err != nil {
			return err
		}
		if appliedFound && appliedState.ProfileID == profileID {
			return fmt.Errorf("profile %d is currently applied; restore vanilla before deleting it", profileID)
		}
	}

	return s.store.DeleteProfile(ctx, profileID)
}

func (s *ProfileService) GetAppliedProfileSummary(ctx context.Context, gameID int64) (summary *AppliedProfileSummary, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get applied profile summary: %w", err)
		}
	}()

	state, found, err := s.store.GetAppliedProfileState(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	profileName := fmt.Sprintf("Profile %d unavailable", state.ProfileID)
	profile, profileFound, err := s.store.GetProfile(ctx, state.ProfileID)
	if err != nil {
		return nil, err
	}
	if profileFound {
		profileName = profile.Name
	}

	hasChanged, err := s.hasAppliedProfileCompositionChanged(ctx, state)
	if err != nil {
		return nil, err
	}

	return &AppliedProfileSummary{
		GameID:                   state.GameID,
		ProfileID:                state.ProfileID,
		ProfileName:              profileName,
		AppliedAt:                state.AppliedAt,
		HasAppliedProfileChanged: hasChanged,
	}, nil
}

func (s *ProfileService) hasAppliedProfileCompositionChanged(ctx context.Context, state storage.AppliedProfileState) (*bool, error) {
	if state.ProfileCompositionSnapshotJSON == nil || state.ProfileCompositionSnapshotHash == nil {
		return nil, nil
	}
	if *state.ProfileCompositionSnapshotJSON == "" || *state.ProfileCompositionSnapshotHash == "" {
		return nil, nil
	}

	profileMods, err := s.store.ListProfileMods(ctx, state.ProfileID)
	if err != nil {
		return nil, err
	}
	currentSnapshot, err := encodeProfileCompositionSnapshot(state.ProfileID, profileMods)
	if err != nil {
		return nil, err
	}

	changed := currentSnapshot.Hash != *state.ProfileCompositionSnapshotHash
	return &changed, nil
}

func encodeProfileCompositionSnapshot(profileID int64, profileMods []storage.ProfileMod) (appliedstate.EncodedProfileCompositionSnapshot, error) {
	mods := make([]appliedstate.ProfileCompositionMod, 0, len(profileMods))
	for _, profileMod := range profileMods {
		mods = append(mods, appliedstate.ProfileCompositionMod{
			ModID:     profileMod.ModID,
			Enabled:   profileMod.Enabled,
			LoadOrder: profileMod.LoadOrder,
		})
	}

	return appliedstate.EncodeProfileCompositionSnapshot(appliedstate.BuildProfileCompositionDocument(profileID, mods))
}
