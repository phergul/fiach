package appliedstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestEncodeProfileCompositionSnapshotHashesCompositionOnly(t *testing.T) {
	base := []ProfileCompositionMod{
		{ModID: 20, Enabled: false, LoadOrder: 1},
		{ModID: 10, Enabled: true, LoadOrder: 0},
	}

	snapshot, err := EncodeProfileCompositionSnapshot(BuildProfileCompositionDocument(5, base))
	if err != nil {
		t.Fatalf("EncodeProfileCompositionSnapshot() error = %v", err)
	}
	if snapshot.JSON == "" || snapshot.Hash == "" {
		t.Fatalf("EncodeProfileCompositionSnapshot() = %+v, want JSON and hash", snapshot)
	}
	if snapshot.Hash != sha256Hex(snapshot.JSON) {
		t.Fatalf("composition snapshot hash = %q, want SHA-256 of JSON", snapshot.Hash)
	}

	var decoded ProfileCompositionDocument
	if err := json.Unmarshal([]byte(snapshot.JSON), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.Version != DocumentVersion || decoded.ProfileID != 5 || len(decoded.Mods) != 2 {
		t.Fatalf("decoded composition snapshot = %+v, want versioned profile composition", decoded)
	}
	if decoded.Mods[0].ModID != 10 || !decoded.Mods[0].Enabled || decoded.Mods[0].LoadOrder != 0 || decoded.Mods[1].ModID != 20 || decoded.Mods[1].Enabled || decoded.Mods[1].LoadOrder != 1 {
		t.Fatalf("decoded composition mods = %+v, want load-order sorted composition", decoded.Mods)
	}
}

func TestProfileCompositionSnapshotHashChangesForCompositionChanges(t *testing.T) {
	t.Parallel()

	base := compositionHash(t, []ProfileCompositionMod{
		{ModID: 10, Enabled: true, LoadOrder: 0},
		{ModID: 20, Enabled: true, LoadOrder: 1},
	})

	tests := []struct {
		name string
		mods []ProfileCompositionMod
	}{
		{
			name: "added mod",
			mods: []ProfileCompositionMod{
				{ModID: 10, Enabled: true, LoadOrder: 0},
				{ModID: 20, Enabled: true, LoadOrder: 1},
				{ModID: 30, Enabled: true, LoadOrder: 2},
			},
		},
		{
			name: "removed mod",
			mods: []ProfileCompositionMod{
				{ModID: 10, Enabled: true, LoadOrder: 0},
			},
		},
		{
			name: "disabled mod",
			mods: []ProfileCompositionMod{
				{ModID: 10, Enabled: false, LoadOrder: 0},
				{ModID: 20, Enabled: true, LoadOrder: 1},
			},
		},
		{
			name: "reordered mods",
			mods: []ProfileCompositionMod{
				{ModID: 10, Enabled: true, LoadOrder: 1},
				{ModID: 20, Enabled: true, LoadOrder: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compositionHash(t, tt.mods); got == base {
				t.Fatalf("compositionHash(%s) = base hash, want changed hash", tt.name)
			}
		})
	}
}

func TestProfileCompositionSnapshotHashIgnoresInputOrderAndNames(t *testing.T) {
	t.Parallel()

	base := compositionHash(t, []ProfileCompositionMod{
		{ModID: 10, Enabled: true, LoadOrder: 0},
		{ModID: 20, Enabled: false, LoadOrder: 1},
	})
	reorderedInput := compositionHash(t, []ProfileCompositionMod{
		{ModID: 20, Enabled: false, LoadOrder: 1},
		{ModID: 10, Enabled: true, LoadOrder: 0},
	})

	if reorderedInput != base {
		t.Fatalf("composition hash changed for input order only: %q != %q", reorderedInput, base)
	}
}

func compositionHash(t *testing.T, mods []ProfileCompositionMod) string {
	t.Helper()

	snapshot, err := EncodeProfileCompositionSnapshot(BuildProfileCompositionDocument(5, mods))
	if err != nil {
		t.Fatalf("EncodeProfileCompositionSnapshot() error = %v", err)
	}

	return snapshot.Hash
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
