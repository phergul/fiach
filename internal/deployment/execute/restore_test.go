package execute_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment/execute"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/fileops"
)

func TestExecuteRestoreRemovesAddedFilesAndRestoresReplacedFiles(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	addedPath := writeRestoreTestFile(t, gameRoot, "Data/added.txt", "added")
	targetPath := writeRestoreTestFile(t, gameRoot, "Data/replaced.txt", "modded")
	backupPath := writeRestoreTestFile(t, storageRoot, "deployment-backups/Data/replaced.txt", "vanilla")
	createdDirectory := filepath.Join(gameRoot, "Mods", "Created")
	if err := os.MkdirAll(createdDirectory, 0o755); err != nil {
		t.Fatalf("MkdirAll() created directory error = %v", err)
	}

	addedSHA256, addedSize, err := fileops.FileIntegrity(addedPath)
	if err != nil {
		t.Fatalf("FileIntegrity() added error = %v", err)
	}
	moddedSHA256, moddedSize, err := fileops.FileIntegrity(targetPath)
	if err != nil {
		t.Fatalf("FileIntegrity() modded error = %v", err)
	}
	baselineSHA256, baselineSize, err := fileops.FileIntegrity(backupPath)
	if err != nil {
		t.Fatalf("FileIntegrity() backup error = %v", err)
	}

	states := []appliedstate.PersistedFileState{
		{
			GameRelativePath: "Data/added.txt",
			BaselineExists:   false,
			AppliedExists:    true,
			AppliedSHA256:    &addedSHA256,
			AppliedSizeBytes: &addedSize,
			OutputKind:       appliedstate.OutputKindCopied,
		},
		{
			GameRelativePath:   "Data/replaced.txt",
			BaselineExists:     true,
			BaselineSHA256:     &baselineSHA256,
			BaselineSizeBytes:  &baselineSize,
			BaselineBackupPath: &backupPath,
			AppliedExists:      true,
			AppliedSHA256:      &moddedSHA256,
			AppliedSizeBytes:   &moddedSize,
			OutputKind:         appliedstate.OutputKindCopied,
		},
	}

	plan, err := planner.PlanRestorePreview(states, gameRoot)
	if err != nil {
		t.Fatalf("PlanRestorePreview() error = %v", err)
	}

	result, err := execute.ExecuteRestore(context.Background(), execute.RestoreContext{
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
		Plan:               plan,
		CreatedDirectories: []execute.RestoreCreatedDirectory{
			{GameRelativePath: "Mods/Created"},
		},
	})
	if err != nil {
		t.Fatalf("ExecuteRestore() error = %v", err)
	}
	if !result.Success || result.CompletedCount < 2 {
		t.Fatalf("ExecuteRestore() = %+v, want successful restore", result)
	}

	assertRestoreTestPathMissing(t, addedPath)
	assertRestoreTestFileContents(t, targetPath, "vanilla")
	assertRestoreTestPathMissing(t, backupPath)
	assertRestoreTestPathMissing(t, createdDirectory)
}

func writeRestoreTestFile(t *testing.T, root string, relativePath string, contents string) string {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	return path
}

func assertRestoreTestFileContents(t *testing.T, path string, want string) {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if string(contents) != want {
		t.Fatalf("ReadFile(%q) = %q, want %q", path, contents, want)
	}
}

func assertRestoreTestPathMissing(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) err = %v, want not exist", path, err)
	}
}
