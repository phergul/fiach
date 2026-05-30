package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
)

func (s *ProfileService) ListProfileMods(ctx context.Context, profileID int64) (mods []dto.ProfileMod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list profile mods: %w", err)
		}
	}()

	profileMods, err := s.store.ListProfileMods(ctx, profileID)
	if err != nil {
		return nil, err
	}

	return mappers.ToDTOProfileMods(profileMods), nil
}

func (s *ProfileService) AddModToProfile(ctx context.Context, profileID int64, modID int64) (profileMod dto.ProfileMod, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationAddProfileMod, "Profile mod add started",
		slog.Int64("profile_id", profileID),
		slog.Int64("mod_id", modID),
	)
	defer func() {
		if err != nil {
			diag.fail("Profile mod add failed", err)
			err = fmt.Errorf("add mod to profile: %w", err)
		}
	}()

	storedProfileMod, err := s.store.AddModToProfile(ctx, profileID, modID)
	if err != nil {
		return dto.ProfileMod{}, err
	}

	profileMod = mappers.ToDTOProfileMod(storedProfileMod)
	diag.complete("Profile mod add completed",
		slog.Bool("enabled", storedProfileMod.Enabled),
		slog.Int64("load_order", storedProfileMod.LoadOrder),
	)

	return profileMod, nil
}

func (s *ProfileService) RemoveModFromProfile(ctx context.Context, profileID int64, modID int64) (err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationRemoveProfileMod, "Profile mod remove started",
		slog.Int64("profile_id", profileID),
		slog.Int64("mod_id", modID),
	)
	defer func() {
		if err != nil {
			diag.fail("Profile mod remove failed", err)
			err = fmt.Errorf("remove mod from profile: %w", err)
		}
	}()

	if err := s.store.RemoveModFromProfile(ctx, profileID, modID); err != nil {
		return err
	}

	diag.complete("Profile mod remove completed")

	return nil
}

func (s *ProfileService) SetProfileModEnabled(ctx context.Context, profileID int64, modID int64, enabled bool) (profileMod dto.ProfileMod, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationSetProfileModEnabled, "Profile mod enabled update started",
		slog.Int64("profile_id", profileID),
		slog.Int64("mod_id", modID),
		slog.Bool("enabled", enabled),
	)
	defer func() {
		if err != nil {
			diag.fail("Profile mod enabled update failed", err)
			err = fmt.Errorf("set profile mod enabled: %w", err)
		}
	}()

	storedProfileMod, err := s.store.SetProfileModEnabled(ctx, profileID, modID, enabled)
	if err != nil {
		return dto.ProfileMod{}, err
	}

	profileMod = mappers.ToDTOProfileMod(storedProfileMod)
	diag.complete("Profile mod enabled update completed",
		slog.Int64("load_order", storedProfileMod.LoadOrder),
	)

	return profileMod, nil
}

func (s *ProfileService) ReorderProfileMods(ctx context.Context, profileID int64, modIDs []int64) (mods []dto.ProfileMod, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationReorderProfileMods, "Profile mods reorder started",
		slog.Int64("profile_id", profileID),
		slog.Int("mod_count", len(modIDs)),
	)
	defer func() {
		if err != nil {
			diag.fail("Profile mods reorder failed", err)
			err = fmt.Errorf("reorder profile mods: %w", err)
		}
	}()

	profileMods, err := s.store.ReorderProfileMods(ctx, profileID, modIDs)
	if err != nil {
		return nil, err
	}

	mods = mappers.ToDTOProfileMods(profileMods)
	diag.complete("Profile mods reorder completed")

	return mods, nil
}
