package appliedstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/phergul/mod-manager/internal/operationplan"
)

func TestEncodeManifestUsesVersionedStableJSONShape(t *testing.T) {
	sourcePath := "/mods/SkyUI/Data/SkyUI.esp"
	targetPath := "/games/Skyrim/Data/SkyUI.esp"
	backupPath := "/managed/operation-backups/Data/SkyUI.esp"
	manifest := operationplan.AppliedOperationManifest{
		AddedFiles: []operationplan.AppliedFileManifestEntry{
			{
				OperationIndex: 1,
				Mod:            operationplan.ModContext{ModID: 10, ModName: "SkyUI"},
				SourcePath:     sourcePath,
				TargetPath:     targetPath,
				SHA256:         "added-sha",
				SizeBytes:      42,
			},
		},
		ReplacedFiles: []operationplan.ReplacedFileManifestEntry{
			{
				OperationIndex: 2,
				Mod:            operationplan.ModContext{ModID: 11, ModName: "Patch"},
				SourcePath:     sourcePath,
				TargetPath:     targetPath,
				SHA256:         "target-sha",
				SizeBytes:      43,
				BackupPath:     backupPath,
				BackupSHA256:   "backup-sha",
			},
		},
		CreatedDirectories: []operationplan.AppliedDirectoryManifestEntry{
			{
				OperationIndex: 0,
				Mod:            operationplan.ModContext{ModID: 10, ModName: "SkyUI"},
				TargetPath:     "/games/Skyrim/Data",
			},
		},
	}

	encoded, err := EncodeManifest(BuildManifestDocument(manifest))
	if err != nil {
		t.Fatalf("EncodeManifest() error = %v", err)
	}

	var decoded ManifestDocument
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.Version != DocumentVersion || len(decoded.AddedFiles) != 1 || len(decoded.ReplacedFiles) != 1 || len(decoded.CreatedDirectories) != 1 {
		t.Fatalf("decoded manifest = %+v, want versioned copied manifest entries", decoded)
	}
	if decoded.AddedFiles[0].SHA256 != "added-sha" || decoded.ReplacedFiles[0].BackupSHA256 != "backup-sha" {
		t.Fatalf("decoded manifest integrity = %+v, %+v, want SHA fields", decoded.AddedFiles[0], decoded.ReplacedFiles[0])
	}
	if decoded.AddedFiles[0].Mod.ID != 10 || decoded.AddedFiles[0].Mod.Name != "SkyUI" {
		t.Fatalf("decoded manifest mod = %+v, want tagged mod document", decoded.AddedFiles[0].Mod)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(encoded), &raw); err != nil {
		t.Fatalf("json.Unmarshal() raw error = %v", err)
	}
	addedFiles := raw["addedFiles"].([]any)
	addedFile := addedFiles[0].(map[string]any)
	mod := addedFile["mod"].(map[string]any)
	if _, found := mod["ModID"]; found {
		t.Fatalf("raw mod JSON = %+v, want no leaked Go field names", mod)
	}
	if mod["id"] != float64(10) || mod["name"] != "SkyUI" {
		t.Fatalf("raw mod JSON = %+v, want id/name fields", mod)
	}
}

func TestDecodeManifestValidatesVersionAndNormalizesSlices(t *testing.T) {
	t.Parallel()

	document, err := DecodeManifest(`{"version":1}`)
	if err != nil {
		t.Fatalf("DecodeManifest() error = %v", err)
	}
	if document.Version != DocumentVersion {
		t.Fatalf("DecodeManifest() version = %d, want %d", document.Version, DocumentVersion)
	}
	if document.AddedFiles == nil || document.ReplacedFiles == nil || document.CreatedDirectories == nil {
		t.Fatalf("DecodeManifest() = %+v, want non-nil slices", document)
	}

	if _, err := DecodeManifest(`{"version":2}`); err == nil {
		t.Fatal("DecodeManifest() error = nil, want unsupported version error")
	}
}

func TestEncodeProfileSnapshotHashesDeterministicOperationShape(t *testing.T) {
	sourcePath := "/mods/SkyUI/Data/SkyUI.esp"
	backupPath := "/managed/operation-backups/Data/SkyUI.esp"
	plan := operationplan.OperationPlan{
		CanApply: true,
		Operations: []operationplan.Operation{
			{
				Type:       operationplan.OperationTypeCopy,
				SourcePath: &sourcePath,
				TargetPath: "/games/Skyrim/Data/SkyUI.esp",
				BackupPath: &backupPath,
				Mod:        operationplan.ModContext{ModID: 10, ModName: "SkyUI"},
			},
		},
	}

	snapshot, err := EncodeProfileSnapshot(BuildProfileSnapshotDocument(plan))
	if err != nil {
		t.Fatalf("EncodeProfileSnapshot() error = %v", err)
	}
	if snapshot.JSON == "" || snapshot.Hash == "" {
		t.Fatalf("EncodeProfileSnapshot() = %+v, want JSON and hash", snapshot)
	}
	if snapshot.Hash != sha256Hex(snapshot.JSON) {
		t.Fatalf("snapshot hash = %q, want SHA-256 of JSON", snapshot.Hash)
	}

	var decoded ProfileSnapshotDocument
	if err := json.Unmarshal([]byte(snapshot.JSON), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.Version != DocumentVersion || !decoded.CanApply || len(decoded.Operations) != 1 {
		t.Fatalf("decoded snapshot = %+v, want one versioned operation", decoded)
	}
	operation := decoded.Operations[0]
	if operation.OperationIndex != 0 || operation.Type != operationplan.OperationTypeCopy || operation.Mod.ID != 10 || operation.Mod.Name != "SkyUI" || operation.SourcePath == nil || *operation.SourcePath != sourcePath || operation.BackupPath == nil || *operation.BackupPath != backupPath {
		t.Fatalf("decoded snapshot operation = %+v, want operation-relevant shape", operation)
	}
}

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
