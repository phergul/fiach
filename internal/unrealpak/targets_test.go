package unrealpak

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDetectTargetsFindsNestedContentPaksWithoutCreatingMods(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	paksPath := filepath.Join(root, "Project", "Content", "Paks")
	if err := os.MkdirAll(paksPath, 0o755); err != nil {
		t.Fatalf("create Paks folder: %v", err)
	}

	result, err := DetectTargets(root)
	if err != nil {
		t.Fatalf("DetectTargets() error = %v", err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0] != "Project/Content/Paks/~mods" {
		t.Fatalf("DetectTargets() candidates = %v, want nested target", result.Candidates)
	}
	if _, err := os.Stat(filepath.Join(paksPath, modsDirectoryName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("~mods stat error = %v, want not created", err)
	}
}

func TestDetectTargetsMatchesContentPaksCaseInsensitively(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Game", "content", "PAKS"), 0o755); err != nil {
		t.Fatalf("create Paks folder: %v", err)
	}

	result, err := DetectTargets(root)
	if err != nil {
		t.Fatalf("DetectTargets() error = %v", err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0] != "Game/content/PAKS/~mods" {
		t.Fatalf("DetectTargets() candidates = %v, want case-preserved target", result.Candidates)
	}
}

func TestDetectTargetsReturnsMultipleCandidatesInStableOrder(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, path := range []string{"Zed/Content/Paks", "Alpha/Content/Paks"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(path)), 0o755); err != nil {
			t.Fatalf("create Paks folder: %v", err)
		}
	}

	result, err := DetectTargets(root)
	if err != nil {
		t.Fatalf("DetectTargets() error = %v", err)
	}
	if strings.Join(result.Candidates, "|") != "Alpha/Content/Paks/~mods|Zed/Content/Paks/~mods" {
		t.Fatalf("DetectTargets() candidates = %v, want stable order", result.Candidates)
	}
}

func TestDetectTargetsWarnsWhenNoContentPaksExists(t *testing.T) {
	t.Parallel()

	result, err := DetectTargets(t.TempDir())
	if err != nil {
		t.Fatalf("DetectTargets() error = %v", err)
	}
	if len(result.Candidates) != 0 || len(result.Warnings) != 1 {
		t.Fatalf("DetectTargets() = %+v, want no candidates and warning", result)
	}
}

func TestDetectTargetsDoesNotFollowDirectorySymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	t.Parallel()

	root := t.TempDir()
	external := filepath.Join(t.TempDir(), "Project", "Content", "Paks")
	if err := os.MkdirAll(external, 0o755); err != nil {
		t.Fatalf("create external Paks folder: %v", err)
	}
	if err := os.Symlink(filepath.Dir(filepath.Dir(external)), filepath.Join(root, "LinkedProject")); err != nil {
		t.Fatalf("create project symlink: %v", err)
	}

	result, err := DetectTargets(root)
	if err != nil {
		t.Fatalf("DetectTargets() error = %v", err)
	}
	if len(result.Candidates) != 0 {
		t.Fatalf("DetectTargets() candidates = %v, want symlink skipped", result.Candidates)
	}
}
