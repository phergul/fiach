package appliedstate

import (
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/deployment"
)

func TestBuildFileStatesFromManifestAddedAndReplacedFiles(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	addedPath := filepath.Join(gameRoot, "Data", "added.esp")
	replacedPath := filepath.Join(gameRoot, "Data", "replaced.esp")
	backupPath := filepath.Join(gameRoot, "backups", "replaced.esp")

	document := ManifestDocument{
		Version: DocumentVersionV1,
		AddedFiles: []AddedFile{
			{
				OperationIndex: 0,
				Mod:            Mod{ID: 10, Name: "SkyUI"},
				TargetPath:     addedPath,
				SHA256:         "added-sha",
				SizeBytes:      11,
			},
		},
		ReplacedFiles: []ReplacedFile{
			{
				OperationIndex:  1,
				Mod:             Mod{ID: 11, Name: "Patch"},
				TargetPath:      replacedPath,
				SHA256:          "applied-sha",
				SizeBytes:       22,
				BackupPath:      backupPath,
				BackupSHA256:    "baseline-sha",
				BackupSizeBytes: 33,
			},
		},
	}

	states, err := BuildFileStatesFromManifest(document, gameRoot, 5, "2026-06-27T00:00:00Z")
	if err != nil {
		t.Fatalf("BuildFileStatesFromManifest() error = %v", err)
	}
	if len(states) != 2 {
		t.Fatalf("BuildFileStatesFromManifest() count = %d, want 2", len(states))
	}

	added := findPersistedFileState(states, "Data/added.esp")
	if added == nil || added.BaselineExists || !added.AppliedExists || added.AppliedSHA256 == nil || *added.AppliedSHA256 != "added-sha" {
		t.Fatalf("added file state = %+v, want applied-only copied file", added)
	}

	replaced := findPersistedFileState(states, "Data/replaced.esp")
	if replaced == nil || !replaced.BaselineExists || !replaced.AppliedExists || replaced.BaselineSHA256 == nil || *replaced.BaselineSHA256 != "baseline-sha" || replaced.AppliedSHA256 == nil || *replaced.AppliedSHA256 != "applied-sha" {
		t.Fatalf("replaced file state = %+v, want baseline and applied hashes", replaced)
	}
}

func TestBuildFileStatesFromManifestPrefersV2FilesMap(t *testing.T) {
	t.Parallel()

	document := ManifestDocument{
		Version: DocumentVersionV2,
		Files: map[string]ManifestFileEntry{
			deployment.CanonicalGameRelativePath("Data/from-map.esp"): {
				GameRelativePath: "Data/from-map.esp",
				OutputKind:       OutputKindCopied,
				AppliedExists:    true,
				AppliedSHA256:    "map-sha",
				AppliedSizeBytes: 99,
				WinningModID:     int64Ptr(10),
			},
		},
	}

	states, err := BuildFileStatesFromManifest(document, t.TempDir(), 5, "2026-06-27T00:00:00Z")
	if err != nil {
		t.Fatalf("BuildFileStatesFromManifest() error = %v", err)
	}
	if len(states) != 1 || states[0].GameRelativePath != "Data/from-map.esp" || states[0].AppliedSHA256 == nil || *states[0].AppliedSHA256 != "map-sha" {
		t.Fatalf("BuildFileStatesFromManifest() = %+v, want v2 files map state", states)
	}
}

func TestAbsoluteToGameRelativePathRejectsPathsOutsideInstallRoot(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	outsidePath := filepath.Join(filepath.Dir(gameRoot), "outside.txt")

	if _, err := AbsoluteToGameRelativePath(gameRoot, outsidePath); err == nil {
		t.Fatal("AbsoluteToGameRelativePath() error = nil, want outside-root error")
	}
}

func TestAttachManifestFilesBuildsCanonicalFilesMap(t *testing.T) {
	t.Parallel()

	appliedSHA256 := "applied-sha"
	document := ManifestDocument{
		Version: DocumentVersionV1,
		AddedFiles: []AddedFile{
			{
				Mod:        Mod{ID: 10, Name: "SkyUI"},
				TargetPath: filepath.Join(t.TempDir(), "Data", "SkyUI.esp"),
				SHA256:     appliedSHA256,
				SizeBytes:  42,
			},
		},
	}

	AttachManifestFiles(&document, []PersistedFileState{
		{
			GameRelativePath: "Data/SkyUI.esp",
			ProfileID:        5,
			AppliedExists:    true,
			AppliedSHA256:    &appliedSHA256,
			OutputKind:       OutputKindCopied,
		},
	})

	if document.Version != DocumentVersionV2 || len(document.Files) != 1 {
		t.Fatalf("AttachManifestFiles() document = %+v, want v2 files map", document)
	}
	entry := document.Files[deployment.CanonicalGameRelativePath("Data/SkyUI.esp")]
	if entry.GameRelativePath != "Data/SkyUI.esp" || entry.AppliedSHA256 != appliedSHA256 {
		t.Fatalf("AttachManifestFiles() entry = %+v, want attached file entry", entry)
	}
}

func TestFileStatesFromStoredManifestMigratesV1Manifest(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "Data", "modded.txt")
	manifestJSON := `{
		"version":1,
		"addedFiles":[{"operationIndex":0,"mod":{"id":10,"name":"Mod"},"sourcePath":"/src","targetPath":"` + filepathToSlash(targetPath) + `","sha256":"added-sha","sizeBytes":5}],
		"replacedFiles":[],
		"createdDirectories":[]
	}`

	states, err := FileStatesFromStoredManifest(manifestJSON, gameRoot, 5, "2026-06-27T00:00:00Z")
	if err != nil {
		t.Fatalf("FileStatesFromStoredManifest() error = %v", err)
	}
	if len(states) != 1 || states[0].GameRelativePath != "Data/modded.txt" {
		t.Fatalf("FileStatesFromStoredManifest() = %+v, want migrated added file", states)
	}
}

func findPersistedFileState(states []PersistedFileState, gameRelativePath string) *PersistedFileState {
	key := deployment.CanonicalGameRelativePath(gameRelativePath)
	for index := range states {
		if deployment.CanonicalGameRelativePath(states[index].GameRelativePath) == key {
			return &states[index]
		}
	}

	return nil
}

func int64Ptr(value int64) *int64 {
	return &value
}
