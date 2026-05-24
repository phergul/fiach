package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/phergul/mod-manager/internal/gamesource"
	"github.com/phergul/mod-manager/internal/services/dto"
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

func (s *GamesService) GetStoredGames(ctx context.Context) (games []dto.StoredGame, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get stored games: %w", err)
		}
	}()

	storedGames, err := s.store.ListStoredGames(ctx)
	if err != nil {
		return nil, err
	}

	return toDTOStoredGames(storedGames), nil
}

func (s *GamesService) ScanAndSaveGames(ctx context.Context) (result dto.SourceScanResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("scan and save games: %w", err)
		}
	}()

	if len(s.sources) == 0 {
		return dto.SourceScanResult{}, errors.New("game sources are not configured")
	}

	for _, source := range s.sources {
		if source == nil {
			return dto.SourceScanResult{}, errors.New("game source is not configured")
		}

		sourceID := strings.TrimSpace(source.Source())
		if sourceID == "" {
			return dto.SourceScanResult{}, errors.New("game source identifier is required")
		}

		sourceGames, err := source.ScanGames(ctx)
		if err != nil {
			return dto.SourceScanResult{}, fmt.Errorf("collect %s games: %w", sourceID, err)
		}

		sourceResult, err := s.store.SaveSourceScan(ctx, sourceID, sourceGames)
		if err != nil {
			return dto.SourceScanResult{}, fmt.Errorf("save %s games: %w", sourceID, err)
		}

		result.Inserted += sourceResult.Inserted
		result.Updated += sourceResult.Updated
		result.MarkedUnavailable += sourceResult.MarkedUnavailable
		result.Games = append(result.Games, toDTOStoredGames(sourceResult.Games)...)
	}

	return result, nil
}
