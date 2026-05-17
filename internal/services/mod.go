package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/phergul/mod-manager/internal/installconfig"
	"github.com/phergul/mod-manager/internal/storage"
)

type ModService struct {
	store *storage.Store
}

func NewModService(store *storage.Store) *ModService {
	return &ModService{
		store: store,
	}
}

func (s *ModService) ListMods(gameID int64) (mods []storage.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list mods: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return nil, errors.New("storage is not configured")
	}

	return s.store.ListMods(context.Background(), gameID)
}

func (s *ModService) ListImportStrategies() (strategies []installconfig.StrategyDescriptor, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list import strategies: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return nil, errors.New("storage is not configured")
	}

	return installconfig.SelectableStrategies(), nil
}
