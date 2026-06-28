package provenance

import (
	"fmt"
	"strings"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/loadorder"
)

func ExplainWinner(
	category deployment.ConflictCategory,
	writers []deployment.WriterEntry,
	winner deployment.WriterEntry,
	ruleApplied bool,
) string {
	if ruleApplied {
		return explainPerFileRuleWinner(winner)
	}

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

func explainPerFileRuleWinner(winner deployment.WriterEntry) string {
	return fmt.Sprintf(
		"Mod %s wins for this path via a saved per-file rule (overrides load order).",
		winner.ModName,
	)
}

func explainExpectedOverwrite(writers []deployment.WriterEntry, winner deployment.WriterEntry) string {
	losers := make([]string, 0, len(writers))
	for _, writer := range writers {
		if writer.SourceKind != deployment.SourceKindMod || writer.IsWinner {
			continue
		}
		losers = append(losers, fmt.Sprintf("%s (load order %d)", writer.ModName, loadorder.DisplayIndex(writer.LoadOrder)))
	}

	if len(losers) == 0 {
		return fmt.Sprintf("Mod %s (load order %d) provides the final file for this path.", winner.ModName, loadorder.DisplayIndex(winner.LoadOrder))
	}

	return fmt.Sprintf(
		"Mod %s (load order %d) wins over %s because later mods override earlier ones for the same path.",
		winner.ModName,
		loadorder.DisplayIndex(winner.LoadOrder),
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
		loadorder.DisplayIndex(loadOrder),
	)
}
