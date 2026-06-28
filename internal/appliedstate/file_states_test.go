package appliedstate

import (
	"path/filepath"
	"testing"
)

func TestAbsoluteToGameRelativePathRejectsPathsOutsideInstallRoot(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	outsidePath := filepath.Join(filepath.Dir(gameRoot), "outside.txt")

	if _, err := AbsoluteToGameRelativePath(gameRoot, outsidePath); err == nil {
		t.Fatal("AbsoluteToGameRelativePath() error = nil, want outside-root error")
	}
}
