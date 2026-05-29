package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/phergul/mod-manager/internal/diagnostics"
	"github.com/phergul/mod-manager/internal/gamesource"
	"github.com/phergul/mod-manager/internal/reshade"
	"github.com/phergul/mod-manager/internal/services/dto"
	"github.com/phergul/mod-manager/internal/services/dto/mappers"
	"github.com/phergul/mod-manager/internal/storage"
)

type GamesService struct {
	store           *storage.Store
	sources         []gamesource.GameSource
	logger          *slog.Logger
	operatingSystem string
}

func NewGamesService(store *storage.Store, logger *slog.Logger, sources ...gamesource.GameSource) *GamesService {
	return &GamesService{
		store:           store,
		sources:         sources,
		logger:          logger,
		operatingSystem: runtime.GOOS,
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

		s.logger.InfoContext(ctx, fmt.Sprintf("%s scan saved", sourceID),
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
		result.Games = append(result.Games, mappers.ToDTOStoredGames(sourceResult.Games)...)
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

func (s *GamesService) DetectGameReShade(ctx context.Context, gameID int64) (result dto.ReShadeDetectionResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("detect game ReShade runtime: %w", err)
		}
	}()

	if s.operatingSystem != "windows" {
		reason := "ReShade runtime detection is only supported on Windows."
		return dto.ReShadeDetectionResult{
			Status:            dto.ReShadeDetectionStatusUnsupported,
			Targets:           []dto.ReShadeTarget{},
			UnsupportedReason: &reason,
		}, nil
	}

	game, err := s.store.GetStoredGame(ctx, gameID)
	if err != nil {
		return dto.ReShadeDetectionResult{}, err
	}

	installPath := strings.TrimSpace(game.InstallPath)
	if installPath == "" {
		return dto.ReShadeDetectionResult{}, errors.New("game install path is required")
	}

	info, err := os.Stat(installPath)
	if err != nil {
		return dto.ReShadeDetectionResult{}, fmt.Errorf("inspect game install path: %w", err)
	}
	if !info.IsDir() {
		return dto.ReShadeDetectionResult{}, fmt.Errorf("game install path %q is not a directory", installPath)
	}

	scanResult, err := reshade.Scan(installPath)
	if err != nil {
		return dto.ReShadeDetectionResult{}, err
	}

	status := dto.ReShadeDetectionStatusNotInstalled
	if len(scanResult.Targets) > 0 {
		status = dto.ReShadeDetectionStatusInstalled
	}

	return dto.ReShadeDetectionResult{
		Status:  status,
		Targets: mappers.ToDTOReShadeTargets(scanResult.Targets),
	}, nil
}
