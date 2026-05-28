package modmetadata

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInventoryParserCountsManagedFilesDirectoriesAndSize(t *testing.T) {
	root := t.TempDir()
	writeInventoryTestFile(t, filepath.Join(root, "Data", "a.txt"), "hello")
	writeInventoryTestFile(t, filepath.Join(root, "Data", "nested", "b.txt"), "world!")

	metadata, err := InventoryParser{}.Parse(context.Background(), ParseInput{ManagedPath: root})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if metadata.FileCount == nil || *metadata.FileCount != 2 {
		t.Fatalf("FileCount = %v, want 2", metadata.FileCount)
	}
	if metadata.DirectoryCount == nil || *metadata.DirectoryCount != 2 {
		t.Fatalf("DirectoryCount = %v, want 2", metadata.DirectoryCount)
	}
	if metadata.TotalSizeBytes == nil || *metadata.TotalSizeBytes != 11 {
		t.Fatalf("TotalSizeBytes = %v, want 11", metadata.TotalSizeBytes)
	}
}

func TestInventoryParserIgnoresIgnoredFilesAndFolders(t *testing.T) {
	root := t.TempDir()
	writeInventoryTestFile(t, filepath.Join(root, "Data", "mod.txt"), "mod")
	writeInventoryTestFile(t, filepath.Join(root, ".DS_Store"), "metadata")
	writeInventoryTestFile(t, filepath.Join(root, "Data", ".DS_Store"), "metadata")

	metadata, err := InventoryParser{}.Parse(context.Background(), ParseInput{ManagedPath: root})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if metadata.FileCount == nil || *metadata.FileCount != 1 {
		t.Fatalf("FileCount = %v, want 1", metadata.FileCount)
	}
	if metadata.DirectoryCount == nil || *metadata.DirectoryCount != 1 {
		t.Fatalf("DirectoryCount = %v, want 1", metadata.DirectoryCount)
	}
	if metadata.TotalSizeBytes == nil || *metadata.TotalSizeBytes != 3 {
		t.Fatalf("TotalSizeBytes = %v, want 3", metadata.TotalSizeBytes)
	}
}

func TestInventoryParserReportsScanFailure(t *testing.T) {
	_, err := InventoryParser{}.Parse(context.Background(), ParseInput{ManagedPath: filepath.Join(t.TempDir(), "missing")})
	if err == nil {
		t.Fatal("Parse() error = nil, want missing path error")
	}
}

func writeInventoryTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
