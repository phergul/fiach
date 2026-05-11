package steam

import (
	"path/filepath"
	"testing"
)

func TestResolveGameImagePathPrefersDirectLibraryArtwork(t *testing.T) {
	t.Parallel()

	artworkRoot := t.TempDir()
	appArtwork := filepath.Join(artworkRoot, "10")
	mkdirAll(t, filepath.Join(appArtwork, "nested"))
	capsulePath := writeArtworkFile(t, appArtwork, "library_capsule.jpg", "capsule")
	nestedPath := writeArtworkFile(t, filepath.Join(appArtwork, "nested"), "library_600x900.jpg", "nested")
	directPath := writeArtworkFile(t, appArtwork, "library_600x900.png", "direct")

	got, err := ResolveGameImagePath(artworkRoot, "10", ImageTypeBanner)
	if err != nil {
		t.Fatalf("ResolveGameImagePath() error = %v", err)
	}

	if got != directPath {
		t.Fatalf("ResolveGameImagePath() = %q, want direct path %q; nested=%q capsule=%q", got, directPath, nestedPath, capsulePath)
	}
}

func TestResolveGameImagePathFallsBackToNestedAndCapsuleArtwork(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filePath []string
	}{
		{
			name:     "nested library artwork",
			filePath: []string{"20", "custom", "library_600x900.jpg"},
		},
		{
			name:     "direct capsule artwork",
			filePath: []string{"20", "library_capsule.png"},
		},
		{
			name:     "nested capsule artwork",
			filePath: []string{"20", "custom", "library_capsule.jpg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			artworkRoot := t.TempDir()
			want := writeArtworkFile(t, filepath.Join(append([]string{artworkRoot}, tt.filePath[:len(tt.filePath)-1]...)...), tt.filePath[len(tt.filePath)-1], "image")

			got, err := ResolveGameImagePath(artworkRoot, "20", ImageTypeBanner)
			if err != nil {
				t.Fatalf("ResolveGameImagePath() error = %v", err)
			}

			if got != want {
				t.Fatalf("ResolveGameImagePath() = %q, want %q", got, want)
			}
		})
	}
}

func TestResolveGameImagePathReturnsEmptyForMissingArtwork(t *testing.T) {
	t.Parallel()

	got, err := ResolveGameImagePath(t.TempDir(), "30", ImageTypeBanner)
	if err != nil {
		t.Fatalf("ResolveGameImagePath() error = %v", err)
	}
	if got != "" {
		t.Fatalf("ResolveGameImagePath() = %q, want empty path", got)
	}
}

func writeArtworkFile(t *testing.T, dir string, name string, content string) string {
	t.Helper()

	mkdirAll(t, dir)
	path := filepath.Join(dir, name)
	writeFile(t, path, content)

	return path
}
