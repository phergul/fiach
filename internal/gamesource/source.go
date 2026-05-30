package gamesource

import (
	"context"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

type GameSource interface {
	Source() string
	ScanGames(ctx context.Context) ([]dbtypes.SourceGame, error)
}
