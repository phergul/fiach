package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/phergul/mod-manager/internal/storage"
)

func (s *ProfileService) ListProfileMods(profileID int64) (mods []storage.ProfileMod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list profile mods: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return nil, errors.New("storage is not configured")
	}

	return s.store.ListProfileMods(context.Background(), profileID)
}

func (s *ProfileService) AddModToProfile(profileID int64, modID int64) (profileMod storage.ProfileMod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("add mod to profile: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return storage.ProfileMod{}, errors.New("storage is not configured")
	}

	return s.store.AddModToProfile(context.Background(), profileID, modID)
}

func (s *ProfileService) RemoveModFromProfile(profileID int64, modID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("remove mod from profile: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return errors.New("storage is not configured")
	}

	return s.store.RemoveModFromProfile(context.Background(), profileID, modID)
}

func (s *ProfileService) SetProfileModEnabled(profileID int64, modID int64, enabled bool) (profileMod storage.ProfileMod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("set profile mod enabled: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return storage.ProfileMod{}, errors.New("storage is not configured")
	}

	return s.store.SetProfileModEnabled(context.Background(), profileID, modID, enabled)
}

func (s *ProfileService) ReorderProfileMods(profileID int64, modIDs []int64) (mods []storage.ProfileMod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("reorder profile mods: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return nil, errors.New("storage is not configured")
	}

	return s.store.ReorderProfileMods(context.Background(), profileID, modIDs)
}
