package provenance

import (
	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
)

func ReconcileModAddedPaths(state *deployment.DesiredState, appliedStates []appliedstate.PersistedFileState) {
	if state == nil || len(appliedStates) == 0 {
		return
	}

	appliedByPath := map[string]appliedstate.PersistedFileState{}
	for _, appliedState := range appliedStates {
		key := deployment.CanonicalGameRelativePath(appliedState.GameRelativePath)
		appliedByPath[key] = appliedState
	}

	for canonicalPath, file := range state.Files {
		appliedState, found := appliedByPath[canonicalPath]
		if !found || !appliedState.AppliedExists || appliedState.BaselineExists {
			continue
		}

		file.Writers = withoutBaseGameWriters(file.Writers)
		file.FileStatus = deployment.FileStatusAdded
		file.Writers = RenumberWriterStack(file.Writers)
		state.Files[canonicalPath] = file
	}
}

func withoutBaseGameWriters(writers []deployment.WriterEntry) []deployment.WriterEntry {
	if len(writers) == 0 {
		return writers
	}

	filtered := make([]deployment.WriterEntry, 0, len(writers))
	for _, writer := range writers {
		if writer.SourceKind == deployment.SourceKindBaseGame {
			continue
		}
		filtered = append(filtered, writer)
	}

	return filtered
}
