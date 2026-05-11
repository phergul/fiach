package services

import (
	"context"
	"errors"
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

func (s *ProfileService) CreateProfile(gameID int64, name string) (profile storage.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("create profile: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return storage.ModProfile{}, errors.New("storage is not configured")
	}

	return s.store.CreateProfile(context.Background(), gameID, name)
}

func (s *ProfileService) ListProfiles(gameID int64) (profiles []storage.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list profiles: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return nil, errors.New("storage is not configured")
	}

	return s.store.ListProfiles(context.Background(), gameID)
}

func (s *ProfileService) RenameProfile(profileID int64, name string) (profile storage.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("rename profile: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return storage.ModProfile{}, errors.New("storage is not configured")
	}

	return s.store.RenameProfile(context.Background(), profileID, name)
}

func (s *ProfileService) DeleteProfile(profileID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("delete profile: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return errors.New("storage is not configured")
	}

	return s.store.DeleteProfile(context.Background(), profileID)
}

func (s *ProfileService) ActivateProfile(gameID int64, profileID int64) (profile storage.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("activate profile: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return storage.ModProfile{}, errors.New("storage is not configured")
	}

	return s.store.ActivateProfile(context.Background(), gameID, profileID)
}

func (s *ProfileService) ClearActiveProfile(gameID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("clear active profile: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return errors.New("storage is not configured")
	}

	return s.store.ClearActiveProfile(context.Background(), gameID)
}

func (s *ProfileService) GetActiveProfile(gameID int64) (profile *storage.ModProfile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get active profile: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return nil, errors.New("storage is not configured")
	}

	active, found, err := s.store.GetActiveProfile(context.Background(), gameID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	return &active, nil
}
