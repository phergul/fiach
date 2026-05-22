package gamesource

import (
	"context"

	"github.com/phergul/mod-manager/internal/storage"
)

type GameSource interface {
	Source() string
	ScanGames(ctx context.Context) ([]storage.SourceGame, error)
}
