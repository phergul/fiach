package provenance

import (
	"fmt"
	"strings"

	"github.com/phergul/fiach/internal/deployment"
)

func ExplainWinner(
	category deployment.ConflictCategory,
	writers []deployment.WriterEntry,
	winner deployment.WriterEntry,
) string {
	switch category {
	case deployment.ConflictAmbiguousOverwrite:
		return explainAmbiguousOverwrite(writers)
	case deployment.ConflictDestructiveFileDirectory:
		return "Multiple mods target this path as both a file and a directory, which cannot be deployed safely."
	case deployment.ConflictExpectedOverwrite:
		return explainExpectedOverwrite(writers, winner)
	default:
		if winner.ModID != nil {
			return fmt.Sprintf("Mod %s provides the final file for this path.", winner.ModName)
		}
		return "This path has a single mod writer."
	}
}

func explainExpectedOverwrite(writers []deployment.WriterEntry, winner deployment.WriterEntry) string {
	losers := make([]string, 0, len(writers))
	for _, writer := range writers {
		if writer.SourceKind != deployment.SourceKindMod || writer.IsWinner {
			continue
		}
		losers = append(losers, fmt.Sprintf("%s (load order %d)", writer.ModName, writer.LoadOrder))
	}

	if len(losers) == 0 {
		return fmt.Sprintf("Mod %s (load order %d) provides the final file for this path.", winner.ModName, winner.LoadOrder)
	}

	return fmt.Sprintf(
		"Mod %s (load order %d) wins over %s because later mods override earlier ones for the same path.",
		winner.ModName,
		winner.LoadOrder,
		strings.Join(losers, ", "),
	)
}

func explainAmbiguousOverwrite(writers []deployment.WriterEntry) string {
	names := make([]string, 0, len(writers))
	var loadOrder int64 = -1
	for _, writer := range writers {
		if writer.SourceKind != deployment.SourceKindMod {
			continue
		}
		if loadOrder < 0 {
			loadOrder = writer.LoadOrder
		}
		names = append(names, writer.ModName)
	}

	return fmt.Sprintf(
		"Mods %s both target this path at load order %d. Resolve load order or add a per-file rule.",
		strings.Join(names, " and "),
		loadOrder,
	)
}
