package provenance

import (
	"sort"
	"strconv"

	"github.com/phergul/fiach/internal/deployment"
)

func FinalizeWriters(writers []deployment.WriterEntry) []deployment.WriterEntry {
	sorted := append([]deployment.WriterEntry{}, writers...)
	sort.SliceStable(sorted, func(i int, j int) bool {
		if sorted[i].LoadOrder != sorted[j].LoadOrder {
			return sorted[i].LoadOrder < sorted[j].LoadOrder
		}
		leftID := int64(0)
		rightID := int64(0)
		if sorted[i].ModID != nil {
			leftID = *sorted[i].ModID
		}
		if sorted[j].ModID != nil {
			rightID = *sorted[j].ModID
		}
		return leftID < rightID
	})

	if len(sorted) == 0 {
		return sorted
	}

	maxLoadOrder := sorted[len(sorted)-1].LoadOrder
	tiedWinners := make([]int, 0)
	for index := range sorted {
		sorted[index].Order = index + 1
		sorted[index].IsWinner = false
		sorted[index].WouldWrite = false
		if sorted[index].LoadOrder == maxLoadOrder {
			tiedWinners = append(tiedWinners, index)
		}
	}

	if len(tiedWinners) == 1 {
		winnerIndex := tiedWinners[0]
		sorted[winnerIndex].IsWinner = true
		for index := range sorted {
			if index != winnerIndex {
				sorted[index].WouldWrite = true
			}
		}
	}

	return sorted
}

func NewModWriter(modID int64, modName string, loadOrder int64) deployment.WriterEntry {
	return deployment.WriterEntry{
		SourceKind: deployment.SourceKindMod,
		SourceID:   "mod:" + strconv.FormatInt(modID, 10),
		ModID:      &modID,
		ModName:    modName,
		LoadOrder:  loadOrder,
	}
}

func NewBaseGameWriter() deployment.WriterEntry {
	return deployment.WriterEntry{
		Order:      0,
		SourceKind: deployment.SourceKindBaseGame,
		SourceID:   "base_game",
		IsWinner:   false,
		WouldWrite: false,
	}
}

func RenumberWriterStack(writers []deployment.WriterEntry) []deployment.WriterEntry {
	for index := range writers {
		writers[index].Order = index + 1
	}
	return writers
}
