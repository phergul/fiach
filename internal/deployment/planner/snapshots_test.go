package planner_test

import (
	"testing"

	"github.com/phergul/fiach/internal/deployment/planner"
)

func TestSnapshotsMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		left  planner.FileStateSnapshot
		right planner.FileStateSnapshot
		want  bool
	}{
		{
			name:  "both missing",
			left:  planner.FileStateSnapshot{Exists: false},
			right: planner.FileStateSnapshot{Exists: false},
			want:  true,
		},
		{
			name:  "left missing right exists",
			left:  planner.FileStateSnapshot{Exists: false},
			right: planner.FileStateSnapshot{Exists: true, SHA256: "abc"},
			want:  false,
		},
		{
			name:  "same hash",
			left:  planner.FileStateSnapshot{Exists: true, SHA256: "abc"},
			right: planner.FileStateSnapshot{Exists: true, SHA256: "abc"},
			want:  true,
		},
		{
			name:  "same hash different case",
			left:  planner.FileStateSnapshot{Exists: true, SHA256: "abc"},
			right: planner.FileStateSnapshot{Exists: true, SHA256: "ABC"},
			want:  true,
		},
		{
			name:  "different hash",
			left:  planner.FileStateSnapshot{Exists: true, SHA256: "abc"},
			right: planner.FileStateSnapshot{Exists: true, SHA256: "def"},
			want:  false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := planner.SnapshotsMatch(testCase.left, testCase.right); got != testCase.want {
				t.Fatalf("SnapshotsMatch() = %v, want %v", got, testCase.want)
			}
		})
	}
}
