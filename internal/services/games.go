package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/gamesource"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
	"github.com/phergul/fiach/internal/storage"
)

type GamesService struct {
	store   *storage.Store
	sources []gamesource.GameSource
	logger  *slog.Logger
}

func NewGamesService(store *storage.Store, logger *slog.Logger, sources ...gamesource.GameSource) *GamesService {
	return &GamesService{
		store:   store,
		sources: sources,
		logger:  logger,
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

	return mappers.ToDTOStoredGames(storedGames), nil
}

func (s *GamesService) ScanAndSaveGames(ctx context.Context) (result dto.SourceScanResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationScanGames, "Game scan started",
		slog.Int("source_count", len(s.sources)),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Game scan failed", err, gamesUserError)
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

		diag.infoEvent("source_saved", fmt.Sprintf("%s scan saved", sourceID),
			slog.String("source", sourceID),
			slog.Int("inserted_count", sourceResult.Inserted),
			slog.Int("updated_count", sourceResult.Updated),
			slog.Int("unavailable_count", sourceResult.MarkedUnavailable),
			slog.Int("game_count", len(sourceResult.Games)),
		)

		result.Inserted += sourceResult.Inserted
		result.Updated += sourceResult.Updated
		result.MarkedUnavailable += sourceResult.MarkedUnavailable
		result.Games = append(result.Games, mappers.ToDTOStoredGames(sourceResult.Games)...)
	}

	diag.complete("Game scan completed",
		slog.Int("inserted_count", result.Inserted),
		slog.Int("updated_count", result.Updated),
		slog.Int("unavailable_count", result.MarkedUnavailable),
		slog.Int("game_count", len(result.Games)),
	)

	return result, nil
}
