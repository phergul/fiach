package loadorder

import "testing"

func TestDisplayIndex(t *testing.T) {
	t.Parallel()

	if got := DisplayIndex(0); got != 1 {
		t.Fatalf("DisplayIndex(0) = %d, want 1", got)
	}
	if got := DisplayIndex(3); got != 4 {
		t.Fatalf("DisplayIndex(3) = %d, want 4", got)
	}
}
