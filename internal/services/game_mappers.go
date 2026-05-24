package services

import (
	"github.com/phergul/mod-manager/internal/services/dto"
	"github.com/phergul/mod-manager/internal/storage/dbtypes"
)

func toDTOStoredGame(game dbtypes.StoredGame) dto.StoredGame {
	return dto.StoredGame(game)
}

func toDTOStoredGames(games []dbtypes.StoredGame) []dto.StoredGame {
	result := make([]dto.StoredGame, 0, len(games))
	for _, game := range games {
		result = append(result, toDTOStoredGame(game))
	}
	return result
}

func toDTOSourceScanResult(result dbtypes.SourceScanResult) dto.SourceScanResult {
	return dto.SourceScanResult{
		Inserted:          result.Inserted,
		Updated:           result.Updated,
		MarkedUnavailable: result.MarkedUnavailable,
		Games:             toDTOStoredGames(result.Games),
	}
}
