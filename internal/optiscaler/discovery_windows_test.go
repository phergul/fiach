//go:build windows

package optiscaler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverCandidatesRanksManagedAndCommonWin64Evidence(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	commonTarget := filepath.Join(root, "Game", "Binaries", "Win64")
	if err := os.MkdirAll(commonTarget, 0o755); err != nil {
		t.Fatalf("mkdir common target: %v", err)
	}
	copyCurrentExecutable(t, filepath.Join(commonTarget, "Game-Win64-Shipping.exe"))
	copyCurrentExecutable(t, filepath.Join(root, "Launcher.exe"))

	candidates, err := DiscoverCandidates(root, []string{filepath.Join("Game", "Binaries", "Win64")})
	if err != nil {
		t.Fatalf("DiscoverCandidates() error = %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("DiscoverCandidates() = %+v, want two candidates", candidates)
	}
	if !candidates[0].Managed || candidates[0].ExecutableName != "Game-Win64-Shipping.exe" {
		t.Fatalf("first candidate = %+v, want managed Win64 shipping executable", candidates[0])
	}
	if candidates[0].Architecture != "x64" {
		t.Fatalf("Architecture = %q, want x64", candidates[0].Architecture)
	}
}
