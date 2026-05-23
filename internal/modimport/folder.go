package modimport

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/phergul/mod-manager/internal/fileignore"
	"github.com/phergul/mod-manager/internal/fileops"
	"github.com/phergul/mod-manager/internal/storage"
)

type FolderSource struct {
	originalPath string
}

func NewFolderSource(sourceFolderPath string) (source FolderSource, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("prepare folder import source: %w", err)
		}
	}()

	originalPath, err := storage.CanonicalModOriginalSourcePath(sourceFolderPath)
	if err != nil {
		return FolderSource{}, err
	}

	return FolderSource{
		originalPath: originalPath,
	}, nil
}

func (s FolderSource) Type() storage.ModSourceType {
	return storage.ModSourceTypeFolder
}

func (s FolderSource) OriginalPath() string {
	return s.originalPath
}

func (s FolderSource) OriginalName() *string {
	return nil
}

func (s FolderSource) SuggestedName() string {
	return folderName(s.originalPath)
}

func (s FolderSource) Validate() error {
	info, err := os.Stat(s.originalPath)
	if err != nil {
		return fmt.Errorf("read source folder: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source path %q is not a folder", s.originalPath)
	}

	entries, err := os.ReadDir(s.originalPath)
	if err != nil {
		return fmt.Errorf("read source folder entries: %w", err)
	}
	if !hasImportableEntry(entries) {
		return fmt.Errorf("source folder %q is empty", s.originalPath)
	}

	return nil
}

func (s FolderSource) Materialize(destinationPath string) error {
	if err := copyImportFolder(s.originalPath, destinationPath); err != nil {
		return fmt.Errorf("copy source folder: %w", err)
	}

	return nil
}

func folderName(path string) string {
	trimmedPath := filepath.Clean(path)
	name := filepath.Base(trimmedPath)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "Imported Mod"
	}

	return name
}

func copyImportFolder(sourcePath string, destinationPath string) error {
	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return fmt.Errorf("read source folder entries: %w", err)
	}

	for _, entry := range entries {
		if fileignore.Has(entry.Name()) {
			continue
		}

		sourceEntryPath := filepath.Join(sourcePath, entry.Name())
		destinationEntryPath := filepath.Join(destinationPath, entry.Name())
		if err := copyImportPath(sourceEntryPath, destinationEntryPath); err != nil {
			return err
		}
	}

	return nil
}

func copyImportPath(sourcePath string, destinationPath string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("read source path %q: %w", sourcePath, err)
	}

	if info.IsDir() {
		if err := os.Mkdir(destinationPath, info.Mode().Perm()); err != nil {
			return fmt.Errorf("create destination folder %q: %w", destinationPath, err)
		}

		entries, err := os.ReadDir(sourcePath)
		if err != nil {
			return fmt.Errorf("read source folder entries %q: %w", sourcePath, err)
		}

		for _, entry := range entries {
			if fileignore.Has(entry.Name()) {
				continue
			}

			if err := copyImportPath(filepath.Join(sourcePath, entry.Name()), filepath.Join(destinationPath, entry.Name())); err != nil {
				return err
			}
		}

		return nil
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("source path %q is not a regular file or folder", sourcePath)
	}

	return copyImportFile(sourcePath, destinationPath, info.Mode().Perm())
}

func hasImportableEntry(entries []os.DirEntry) bool {
	for _, entry := range entries {
		if !fileignore.Has(entry.Name()) {
			return true
		}
	}

	return false
}

func copyImportFile(sourcePath string, destinationPath string, permissions os.FileMode) error {
	if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
		SourcePath: sourcePath,
		TargetPath: destinationPath,
		Mode:       permissions,
		OpenLabel:  "source file",
	}); err != nil {
		return fmt.Errorf("copy source file %q: %w", sourcePath, err)
	}

	return nil
}
