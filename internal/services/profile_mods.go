package services

import (
	"context"
	"fmt"

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
	defer func() {
		if err != nil {
			err = fmt.Errorf("add mod to profile: %w", err)
		}
	}()

	storedProfileMod, err := s.store.AddModToProfile(ctx, profileID, modID)
	if err != nil {
		return dto.ProfileMod{}, err
	}

	return mappers.ToDTOProfileMod(storedProfileMod), nil
}

func (s *ProfileService) RemoveModFromProfile(ctx context.Context, profileID int64, modID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("remove mod from profile: %w", err)
		}
	}()

	return s.store.RemoveModFromProfile(ctx, profileID, modID)
}

func (s *ProfileService) SetProfileModEnabled(ctx context.Context, profileID int64, modID int64, enabled bool) (profileMod dto.ProfileMod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("set profile mod enabled: %w", err)
		}
	}()

	storedProfileMod, err := s.store.SetProfileModEnabled(ctx, profileID, modID, enabled)
	if err != nil {
		return dto.ProfileMod{}, err
	}

	return mappers.ToDTOProfileMod(storedProfileMod), nil
}

func (s *ProfileService) ReorderProfileMods(ctx context.Context, profileID int64, modIDs []int64) (mods []dto.ProfileMod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("reorder profile mods: %w", err)
		}
	}()

	profileMods, err := s.store.ReorderProfileMods(ctx, profileID, modIDs)
	if err != nil {
		return nil, err
	}

	return mappers.ToDTOProfileMods(profileMods), nil
}
