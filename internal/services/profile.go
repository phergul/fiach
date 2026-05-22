package services

import (
	"context"
	"fmt"

	"github.com/phergul/mod-manager/internal/storage"
)

type ProfileService struct {
	store *storage.Store
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

	return s.store.DeleteProfile(ctx, profileID)
}

func (s *ProfileService) ActivateProfile(ctx context.Context, gameID int64, profileID int64) (profile storage.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("activate profile: %w", err)
		}
	}()

	return s.store.ActivateProfile(ctx, gameID, profileID)
}

func (s *ProfileService) DeactivateProfile(ctx context.Context, gameID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("deactivate profile: %w", err)
		}
	}()

	return s.store.DeactivateProfile(ctx, gameID)
}

func (s *ProfileService) GetActiveProfile(ctx context.Context, gameID int64) (profile *storage.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get active profile: %w", err)
		}
	}()

	active, found, err := s.store.GetActiveProfile(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	return &active, nil
}
