package mappers

import (
	"github.com/phergul/mod-manager/internal/reshade"
	"github.com/phergul/mod-manager/internal/services/dto"
	"github.com/phergul/mod-manager/internal/storage/dbtypes"
)

func ToDTOStoredGame(game dbtypes.StoredGame) dto.StoredGame {
	return dto.StoredGame(game)
}

func ToDTOStoredGames(games []dbtypes.StoredGame) []dto.StoredGame {
	result := make([]dto.StoredGame, 0, len(games))
	for _, game := range games {
		result = append(result, ToDTOStoredGame(game))
	}
	return result
}

func ToDTOSourceScanResult(result dbtypes.SourceScanResult) dto.SourceScanResult {
	return dto.SourceScanResult{
		Inserted:          result.Inserted,
		Updated:           result.Updated,
		MarkedUnavailable: result.MarkedUnavailable,
		Games:             ToDTOStoredGames(result.Games),
	}
}

func ToDTOReShadeTargets(targets []reshade.Target) []dto.ReShadeTarget {
	result := make([]dto.ReShadeTarget, 0, len(targets))
	for _, target := range targets {
		result = append(result, dto.ReShadeTarget{
			Path:        target.Path,
			Executables: append([]string(nil), target.Executables...),
		})
	}

	return result
}
