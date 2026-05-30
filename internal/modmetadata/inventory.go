package modmetadata

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/phergul/fiach/internal/fileignore"
)

type InventoryParser struct{}

func (InventoryParser) Parse(ctx context.Context, input ParseInput) (metadata Metadata, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("parse inventory metadata: %w", err)
		}
	}()

	var fileCount int64
	var directoryCount int64
	var totalSizeBytes int64

	err = filepath.WalkDir(input.ManagedPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == input.ManagedPath {
			return nil
		}
		if fileignore.Has(entry.Name()) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("managed metadata path %q is a symlink", path)
		}
		if entry.IsDir() {
			directoryCount++
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("managed metadata path %q is not a regular file or folder", path)
		}

		fileCount++
		totalSizeBytes += info.Size()
		return nil
	})
	if err != nil {
		return Metadata{}, err
	}

	return Metadata{
		FileCount:      int64Ptr(fileCount),
		DirectoryCount: int64Ptr(directoryCount),
		TotalSizeBytes: int64Ptr(totalSizeBytes),
	}, nil
}
