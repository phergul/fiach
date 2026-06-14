package optiscaler

import (
	"path/filepath"
	"testing"
)

func TestResolveWithinRootRejectsEscape(t *testing.T) {
	t.Parallel()
	if _, err := ResolveWithinRoot(t.TempDir(), filepath.Join("..", "outside")); err == nil {
		t.Fatal("ResolveWithinRoot() error = nil, want escape rejection")
	}
}
