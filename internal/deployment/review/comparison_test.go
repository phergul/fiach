package review

import (
	"testing"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
)

func TestBuildStateComparison(t *testing.T) {
	t.Parallel()

	applied := planner.FileStateSnapshot{Exists: true, SHA256: "applied"}
	current := planner.FileStateSnapshot{Exists: true, SHA256: "current"}
	desired := planner.FileStateSnapshot{Exists: true, SHA256: "desired"}

	comparison := buildStateComparison(applied, current, desired)
	if comparison.AppliedMatchesCurrent {
		t.Fatal("AppliedMatchesCurrent = true, want false")
	}
	if comparison.AppliedMatchesDesired {
		t.Fatal("AppliedMatchesDesired = true, want false")
	}
	if comparison.CurrentMatchesDesired {
		t.Fatal("CurrentMatchesDesired = true, want false")
	}

	allMatch := buildStateComparison(applied, applied, applied)
	if !allMatch.AppliedMatchesCurrent || !allMatch.AppliedMatchesDesired || !allMatch.CurrentMatchesDesired {
		t.Fatalf("buildStateComparison() all match = %+v, want all true", allMatch)
	}
}

func TestBuildDriftExplanation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		driftKind  deployment.DriftKind
		comparison StateComparison
		fileStatus deployment.FileStatus
		want       string
	}{
		{
			name:      "modified drift",
			driftKind: deployment.DriftModified,
			want:      "This file was modified on disk since the last apply.",
		},
		{
			name:      "missing drift",
			driftKind: deployment.DriftMissing,
			want:      "This file is missing from disk but was present after the last apply.",
		},
		{
			name:      "external drift",
			driftKind: deployment.DriftExternal,
			want:      "This file was kept as an external edit and will not be overwritten automatically.",
		},
		{
			name: "profile change without drift",
			comparison: StateComparison{
				AppliedMatchesCurrent: true,
				AppliedMatchesDesired: false,
				CurrentMatchesDesired: false,
			},
			fileStatus: deployment.FileStatusAdded,
			want:       "",
		},
		{
			name: "all match",
			comparison: StateComparison{
				AppliedMatchesCurrent: true,
				AppliedMatchesDesired: true,
				CurrentMatchesDesired: true,
			},
			want: "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := buildDriftExplanation(testCase.driftKind, testCase.comparison, testCase.fileStatus)
			if got != testCase.want {
				t.Fatalf("buildDriftExplanation() = %q, want %q", got, testCase.want)
			}
		})
	}
}
