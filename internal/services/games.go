package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/phergul/mod-manager/internal/diagnostics"
	"github.com/phergul/mod-manager/internal/gamesource"
	"github.com/phergul/mod-manager/internal/services/dto"
	"github.com/phergul/mod-manager/internal/storage"
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

	return toDTOStoredGames(storedGames), nil
}

func (s *GamesService) ScanAndSaveGames(ctx context.Context) (result dto.SourceScanResult, err error) {
	startedAt := time.Now()
	defer func() {
		if err != nil {
			s.logger.ErrorContext(ctx, "Game scan failed",
				slog.String("operation", diagnostics.OperationScanGames),
				slog.String("event", diagnostics.EventFailed),
				diagnostics.DurationAttr(startedAt),
				diagnostics.ErrorAttr(err),
			)
			err = fmt.Errorf("scan and save games: %w", err)
		}
	}()

	s.logger.InfoContext(ctx, "Game scan started",
		slog.String("operation", diagnostics.OperationScanGames),
		slog.String("event", diagnostics.EventStarted),
		slog.Int("source_count", len(s.sources)),
	)

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

		s.logger.InfoContext(ctx, "Game source scan saved",
			slog.String("operation", diagnostics.OperationScanGames),
			slog.String("event", "source_saved"),
			slog.String("source", sourceID),
			slog.Int("inserted_count", sourceResult.Inserted),
			slog.Int("updated_count", sourceResult.Updated),
			slog.Int("unavailable_count", sourceResult.MarkedUnavailable),
			slog.Int("game_count", len(sourceResult.Games)),
		)

		result.Inserted += sourceResult.Inserted
		result.Updated += sourceResult.Updated
		result.MarkedUnavailable += sourceResult.MarkedUnavailable
		result.Games = append(result.Games, toDTOStoredGames(sourceResult.Games)...)
	}

	s.logger.InfoContext(ctx, "Game scan completed",
		slog.String("operation", diagnostics.OperationScanGames),
		slog.String("event", diagnostics.EventCompleted),
		slog.Int("inserted_count", result.Inserted),
		slog.Int("updated_count", result.Updated),
		slog.Int("unavailable_count", result.MarkedUnavailable),
		slog.Int("game_count", len(result.Games)),
		diagnostics.DurationAttr(startedAt),
	)

	return result, nil
}
