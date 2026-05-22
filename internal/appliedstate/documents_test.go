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

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
