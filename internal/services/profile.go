package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
	"github.com/phergul/fiach/internal/storage"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

type ProfileService struct {
	store  *storage.Store
	logger *slog.Logger
}

func NewProfileService(store *storage.Store, logger *slog.Logger) *ProfileService {
	return &ProfileService{
		store:  store,
		logger: logger,
	}
}

func (s *ProfileService) CreateProfile(ctx context.Context, gameID int64, name string) (profile dto.ModProfile, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationCreateProfile, "Profile create started",
		slog.Int64("game_id", gameID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Profile create failed", err, profileUserError)
		}
	}()

	storedProfile, err := s.store.CreateProfile(ctx, gameID, name)
	if err != nil {
		return dto.ModProfile{}, err
	}

	profile = mappers.ToDTOModProfile(storedProfile)
	diag.complete("Profile create completed",
		slog.Int64("profile_id", storedProfile.ID),
		slog.String("profile_name", storedProfile.Name),
	)

	return profile, nil
}

func (s *ProfileService) DuplicateProfile(ctx context.Context, profileID int64) (profile dto.ModProfile, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationDuplicateProfile, "Profile duplicate started",
		slog.Int64("profile_id", profileID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Profile duplicate failed", err, profileUserError)
		}
	}()

	if sourceProfile, found, lookupErr := s.store.GetProfile(ctx, profileID); lookupErr == nil && found {
		diag.attrs = append(diag.attrs, slog.String("source_profile_name", sourceProfile.Name))
	}

	storedProfile, err := s.store.DuplicateProfile(ctx, profileID)
	if err != nil {
		return dto.ModProfile{}, err
	}

	profile = mappers.ToDTOModProfile(storedProfile)
	diag.complete("Profile duplicate completed",
		slog.Int64("profile_id", storedProfile.ID),
		slog.String("profile_name", storedProfile.Name),
	)

	return profile, nil
}

func (s *ProfileService) ListProfiles(ctx context.Context, gameID int64) (profiles []dto.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list profiles: %w", err)
		}
	}()

	storedProfiles, err := s.store.ListProfiles(ctx, gameID)
	if err != nil {
		return nil, err
	}

	return mappers.ToDTOModProfiles(storedProfiles), nil
}

func (s *ProfileService) RenameProfile(ctx context.Context, profileID int64, name string) (profile dto.ModProfile, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationRenameProfile, "Profile rename started",
		slog.Int64("profile_id", profileID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Profile rename failed", err, profileUserError)
		}
	}()

	storedProfile, err := s.store.RenameProfile(ctx, profileID, name)
	if err != nil {
		return dto.ModProfile{}, err
	}

	profile = mappers.ToDTOModProfile(storedProfile)
	diag.complete("Profile rename completed",
		slog.Int64("game_id", storedProfile.GameID),
		slog.String("profile_name", storedProfile.Name),
	)

	return profile, nil
}

func (s *ProfileService) DeleteProfile(ctx context.Context, profileID int64) (err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationDeleteProfile, "Profile delete started",
		slog.Int64("profile_id", profileID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Profile delete failed", err, profileUserError)
		}
	}()

	profile, found, err := s.store.GetProfile(ctx, profileID)
	if err != nil {
		return err
	}
	if found {
		diag.attrs = append(diag.attrs,
			slog.Int64("game_id", profile.GameID),
			slog.String("profile_name", profile.Name),
		)
		appliedState, appliedFound, err := s.store.GetAppliedProfileState(ctx, profile.GameID)
		if err != nil {
			return err
		}
		if appliedFound && appliedState.ProfileID == profileID {
			return apperror.New("Restore vanilla before deleting an applied profile.")
		}
	}

	if err := s.store.DeleteProfile(ctx, profileID); err != nil {
		return err
	}

	diag.complete("Profile delete completed")

	return nil
}

func (s *ProfileService) GetAppliedProfileSummary(ctx context.Context, gameID int64) (summary *dto.AppliedProfileSummary, err error) {
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

	return &dto.AppliedProfileSummary{
		GameID:                   state.GameID,
		ProfileID:                state.ProfileID,
		ProfileName:              profileName,
		AppliedAt:                state.AppliedAt,
		HasAppliedProfileChanged: hasChanged,
	}, nil
}

func (s *ProfileService) hasAppliedProfileCompositionChanged(ctx context.Context, state dbtypes.AppliedProfileState) (*bool, error) {
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

func encodeProfileCompositionSnapshot(profileID int64, profileMods []dbtypes.ProfileMod) (appliedstate.EncodedSnapshot, error) {
	mods := make([]appliedstate.ProfileCompositionMod, 0, len(profileMods))
	for _, profileMod := range profileMods {
		mods = append(mods, appliedstate.ProfileCompositionMod{
			ModID:            profileMod.ModID,
			Enabled:          profileMod.Enabled,
			LoadOrder:        profileMod.LoadOrder,
			SourcePath:       profileMod.SourcePath,
			PackageUpdatedAt: profileMod.ModUpdatedAt,
		})
	}

	return appliedstate.EncodeProfileCompositionSnapshot(appliedstate.BuildProfileCompositionDocument(profileID, mods))
}
