package gamesource

import (
	"context"

	"github.com/phergul/mod-manager/internal/storage/dbtypes"
)

type GameSource interface {
	Source() string
	ScanGames(ctx context.Context) ([]dbtypes.SourceGame, error)
}
