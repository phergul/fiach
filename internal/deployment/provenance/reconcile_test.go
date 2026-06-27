package provenance_test

import (
	"testing"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/provenance"
)

func TestReconcileModAddedPathsRemovesBaseGameWriter(t *testing.T) {
	t.Parallel()

	state := deployment.DesiredState{
		Files: map[string]deployment.DesiredFile{
			deployment.CanonicalGameRelativePath("Screenshots/recording.mov"): {
				GameRelativePath: "Screenshots/recording.mov",
				FileStatus:       deployment.FileStatusReplaced,
				Writers: []deployment.WriterEntry{
					provenance.NewBaseGameWriter(),
					{
						Order:      2,
						SourceKind: deployment.SourceKindMod,
						SourceID:   "mod:1",
						ModName:    "Screenshots",
						LoadOrder:  4,
						IsWinner:   true,
					},
				},
			},
		},
	}

	provenance.ReconcileModAddedPaths(&state, []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Screenshots/recording.mov",
			AppliedExists:    true,
			BaselineExists:   false,
		},
	})

	file := state.Files[deployment.CanonicalGameRelativePath("Screenshots/recording.mov")]
	if file.FileStatus != deployment.FileStatusAdded {
		t.Fatalf("file status = %q, want added", file.FileStatus)
	}
	if len(file.Writers) != 1 {
		t.Fatalf("writers = %+v, want single mod writer", file.Writers)
	}
	if file.Writers[0].SourceKind != deployment.SourceKindMod || file.Writers[0].ModName != "Screenshots" {
		t.Fatalf("writers = %+v, want Screenshots mod writer", file.Writers)
	}
	if file.Writers[0].Order != 1 {
		t.Fatalf("writer order = %d, want renumbered to 1", file.Writers[0].Order)
	}
}

func TestReconcileModAddedPathsKeepsBaseGameForVanillaReplace(t *testing.T) {
	t.Parallel()

	state := deployment.DesiredState{
		Files: map[string]deployment.DesiredFile{
			deployment.CanonicalGameRelativePath("Data/existing.esp"): {
				GameRelativePath: "Data/existing.esp",
				FileStatus:       deployment.FileStatusReplaced,
				Writers: []deployment.WriterEntry{
					provenance.NewBaseGameWriter(),
					{
						Order:      2,
						SourceKind: deployment.SourceKindMod,
						SourceID:   "mod:1",
						ModName:    "Mod",
						LoadOrder:  0,
						IsWinner:   true,
					},
				},
			},
		},
	}

	provenance.ReconcileModAddedPaths(&state, []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/existing.esp",
			AppliedExists:    true,
			BaselineExists:   true,
		},
	})

	file := state.Files[deployment.CanonicalGameRelativePath("Data/existing.esp")]
	if file.FileStatus != deployment.FileStatusReplaced {
		t.Fatalf("file status = %q, want replaced", file.FileStatus)
	}
	if len(file.Writers) != 2 || file.Writers[0].SourceKind != deployment.SourceKindBaseGame {
		t.Fatalf("writers = %+v, want base_game retained", file.Writers)
	}
}
