package mappers

import (
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage/dbtypes"
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
