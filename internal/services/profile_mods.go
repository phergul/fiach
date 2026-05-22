package services

import (
	"context"
	"fmt"

	"github.com/phergul/mod-manager/internal/storage"
)

func (s *ProfileService) ListProfileMods(ctx context.Context, profileID int64) (mods []storage.ProfileMod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list profile mods: %w", err)
		}
	}()

	return s.store.ListProfileMods(ctx, profileID)
}

func (s *ProfileService) AddModToProfile(ctx context.Context, profileID int64, modID int64) (profileMod storage.ProfileMod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("add mod to profile: %w", err)
		}
	}()

	return s.store.AddModToProfile(ctx, profileID, modID)
}

func (s *ProfileService) RemoveModFromProfile(ctx context.Context, profileID int64, modID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("remove mod from profile: %w", err)
		}
	}()

	return s.store.RemoveModFromProfile(ctx, profileID, modID)
}

func (s *ProfileService) SetProfileModEnabled(ctx context.Context, profileID int64, modID int64, enabled bool) (profileMod storage.ProfileMod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("set profile mod enabled: %w", err)
		}
	}()

	return s.store.SetProfileModEnabled(ctx, profileID, modID, enabled)
}

func (s *ProfileService) ReorderProfileMods(ctx context.Context, profileID int64, modIDs []int64) (mods []storage.ProfileMod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("reorder profile mods: %w", err)
		}
	}()

	return s.store.ReorderProfileMods(ctx, profileID, modIDs)
}
