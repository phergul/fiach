package deployment_test

import (
	"testing"

	"github.com/phergul/fiach/internal/deployment"
)

func TestCanonicalGameRelativePath_CaseFolding(t *testing.T) {
	t.Parallel()

	left := deployment.CanonicalGameRelativePath("Data/Foo.txt")
	right := deployment.CanonicalGameRelativePath("data/foo.txt")
	if left != right {
		t.Fatalf("canonical paths = %q and %q, want equal", left, right)
	}
	if left != "data/foo.txt" {
		t.Fatalf("canonical path = %q, want data/foo.txt", left)
	}
}

func TestIsStrictPathPrefix(t *testing.T) {
	t.Parallel()

	if !deployment.IsStrictPathPrefix("shared", "shared/plugin.txt") {
		t.Fatal("shared should be a strict prefix of shared/plugin.txt")
	}
	if deployment.IsStrictPathPrefix("shared", "shared") {
		t.Fatal("shared should not be a strict prefix of itself")
	}
	if deployment.IsStrictPathPrefix("shared/plugin.txt", "shared") {
		t.Fatal("shared/plugin.txt should not prefix shared")
	}
}
