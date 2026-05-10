package steam

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSteamRoot(t *testing.T) {
	t.Parallel()

	root := createSteamRoot(t)

	got, err := ValidateSteamRoot(root)
	if err != nil {
		t.Fatalf("ValidateSteamRoot() error = %v", err)
	}

	if got.Root != filepath.Clean(root) {
		t.Fatalf("Root = %q, want %q", got.Root, filepath.Clean(root))
	}
	if got.SteamApps != filepath.Join(root, "steamapps") {
		t.Fatalf("SteamApps = %q, want steamapps path", got.SteamApps)
	}
	if got.LibraryVDF != filepath.Join(root, "steamapps", "libraryfolders.vdf") {
		t.Fatalf("LibraryVDF = %q, want libraryfolders.vdf path", got.LibraryVDF)
	}
	if got.UserData != filepath.Join(root, "userdata") {
		t.Fatalf("UserData = %q, want userdata path", got.UserData)
	}
	if got.Artwork != filepath.Join(root, "appcache", "librarycache") {
		t.Fatalf("Artwork = %q, want artwork path", got.Artwork)
	}
}

func TestValidateSteamRootReturnsNotFoundForInvalidLayouts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T) string
	}{
		{
			name: "missing root",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "missing")
			},
		},
		{
			name: "missing steamapps",
			setup: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
		},
		{
			name: "missing library vdf",
			setup: func(t *testing.T) string {
				t.Helper()
				root := t.TempDir()
				mkdirAll(t, filepath.Join(root, "steamapps"))
				mkdirAll(t, filepath.Join(root, "userdata"))
				return root
			},
		},
		{
			name: "missing userdata",
			setup: func(t *testing.T) string {
				t.Helper()
				root := t.TempDir()
				mkdirAll(t, filepath.Join(root, "steamapps"))
				writeFile(t, filepath.Join(root, "steamapps", "libraryfolders.vdf"))
				return root
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ValidateSteamRoot(tt.setup(t))
			if !errors.Is(err, ErrSteamNotFound) {
				t.Fatalf("ValidateSteamRoot() error = %v, want ErrSteamNotFound", err)
			}
		})
	}
}

func TestFindSteamPathsUsesManualPathFirst(t *testing.T) {
	manualRoot := createSteamRoot(t)
	autoRoot := createSteamRoot(t)

	restore := stubSteamRootCandidates(func() []string {
		return []string{autoRoot}
	})
	defer restore()

	got, err := FindSteamPaths(manualRoot)
	if err != nil {
		t.Fatalf("FindSteamPaths() error = %v", err)
	}

	if got.Root != filepath.Clean(manualRoot) {
		t.Fatalf("Root = %q, want manual root %q", got.Root, manualRoot)
	}
}

func TestFindSteamPathsReturnsManualPathErrorWithoutFallback(t *testing.T) {
	autoRoot := createSteamRoot(t)
	invalidManualRoot := filepath.Join(t.TempDir(), "missing")

	restore := stubSteamRootCandidates(func() []string {
		return []string{autoRoot}
	})
	defer restore()

	_, err := FindSteamPaths(invalidManualRoot)
	if !errors.Is(err, ErrSteamNotFound) {
		t.Fatalf("FindSteamPaths() error = %v, want ErrSteamNotFound", err)
	}
}

func TestFindSteamPathsUsesAutoDetectedCandidates(t *testing.T) {
	validRoot := createSteamRoot(t)
	missingRoot := filepath.Join(t.TempDir(), "missing")

	restore := stubSteamRootCandidates(func() []string {
		return []string{missingRoot, validRoot}
	})
	defer restore()

	got, err := FindSteamPaths("")
	if err != nil {
		t.Fatalf("FindSteamPaths() error = %v", err)
	}

	if got.Root != filepath.Clean(validRoot) {
		t.Fatalf("Root = %q, want %q", got.Root, validRoot)
	}
}

func TestFindSteamPathsReturnsNotFoundWhenNoCandidatesAreValid(t *testing.T) {
	restore := stubSteamRootCandidates(func() []string {
		return []string{filepath.Join(t.TempDir(), "missing")}
	})
	defer restore()

	_, err := FindSteamPaths("")
	if !errors.Is(err, ErrSteamNotFound) {
		t.Fatalf("FindSteamPaths() error = %v, want ErrSteamNotFound", err)
	}
}

func createSteamRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "steamapps"))
	mkdirAll(t, filepath.Join(root, "userdata"))
	writeFile(t, filepath.Join(root, "steamapps", "libraryfolders.vdf"))

	return root
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()

	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func stubSteamRootCandidates(fn func() []string) func() {
	previous := steamRootCandidates
	steamRootCandidates = fn

	return func() {
		steamRootCandidates = previous
	}
}
