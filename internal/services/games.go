package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/phergul/mod-manager/internal/services/gamesource"
	"github.com/phergul/mod-manager/internal/storage"
)

type GamesService struct {
	store   *storage.Store
	sources []gamesource.GameSource
}

func NewGamesService(store *storage.Store, sources ...gamesource.GameSource) *GamesService {
	return &GamesService{
		store:   store,
		sources: sources,
	}
}

func (s *GamesService) GetStoredGames() (games []storage.StoredGame, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get stored games: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return nil, errors.New("storage is not configured")
	}

	return s.store.ListStoredGames(context.Background())
}

func (s *GamesService) ScanAndSaveGames() (result storage.SourceScanResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("scan and save games: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return result, errors.New("storage is not configured")
	}
	if len(s.sources) == 0 {
		return result, errors.New("game sources are not configured")
	}

	for _, source := range s.sources {
		if source == nil {
			return result, errors.New("game source is not configured")
		}

		sourceID := strings.TrimSpace(source.Source())
		if sourceID == "" {
			return result, errors.New("game source identifier is required")
		}

		sourceGames, err := source.ScanGames(context.Background())
		if err != nil {
			return result, fmt.Errorf("collect %s games: %w", sourceID, err)
		}

		sourceResult, err := s.store.SaveSourceScan(context.Background(), sourceID, sourceGames)
		if err != nil {
			return result, fmt.Errorf("save %s games: %w", sourceID, err)
		}

		result.Inserted += sourceResult.Inserted
		result.Updated += sourceResult.Updated
		result.MarkedUnavailable += sourceResult.MarkedUnavailable
		result.Games = append(result.Games, sourceResult.Games...)
	}

	return result, nil
}
