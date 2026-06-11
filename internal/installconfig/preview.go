package installconfig

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/phergul/fiach/internal/fileignore"
	"github.com/phergul/fiach/internal/installpath"
	"github.com/phergul/fiach/internal/unrealpak"
)

const DefaultPreviewFileCap = 100

type PreviewInput struct {
	SourcePath         string
	StrategyType       StrategyType
	TargetRelativePath string
	FileCap            int
}

type Preview struct {
	StrategyType        StrategyType
	TargetBase          string
	TargetRelativePath  string
	TargetDisplayPath   string
	TotalFileCount      int
	TotalDirectoryCount int
	TotalSizeBytes      int64
	TargetFilePaths     []string
	IsCapped            bool
	Cap                 int
	Warnings            []string
}

func BuildPreview(input PreviewInput) (preview Preview, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("build import configuration preview: %w", err)
		}
	}()

	if err := ValidateSelectableStrategy(input.StrategyType); err != nil {
		return Preview{}, err
	}
	targetRelativePath, err := NormalizeTargetRelativePath(input.TargetRelativePath)
	if err != nil {
		return Preview{}, err
	}

	fileCap := input.FileCap
	if fileCap <= 0 {
		fileCap = DefaultPreviewFileCap
	}

	preview = Preview{
		StrategyType:       input.StrategyType,
		TargetBase:         TargetBaseGameRoot,
		TargetRelativePath: targetRelativePath,
		TargetDisplayPath:  DisplayTargetRelativePath(targetRelativePath),
		Cap:                fileCap,
		TargetFilePaths:    []string{},
		Warnings:           []string{},
	}

	switch input.StrategyType {
	case StrategyTypeGenericCopy:
		err = buildGenericCopyPreview(input.SourcePath, targetRelativePath, &preview)
	case StrategyTypeUnrealPak:
		err = buildUnrealPakPreview(input.SourcePath, targetRelativePath, &preview)
	default:
		return Preview{}, fmt.Errorf("import strategy %q cannot be previewed", input.StrategyType)
	}
	if err != nil {
		return Preview{}, err
	}

	sort.Strings(preview.TargetFilePaths)
	if len(preview.TargetFilePaths) > fileCap {
		preview.IsCapped = true
		preview.TargetFilePaths = preview.TargetFilePaths[:fileCap]
		preview.Warnings = append(preview.Warnings, fmt.Sprintf("Showing first %d of %d target files.", fileCap, preview.TotalFileCount))
	}

	return preview, nil
}

func buildGenericCopyPreview(sourcePath string, targetRelativePath string, preview *Preview) error {
	return filepath.WalkDir(sourcePath, func(sourceFilePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if sourceFilePath == sourcePath {
			return nil
		}
		if fileignore.Has(entry.Name()) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if entry.IsDir() {
			preview.TotalDirectoryCount++
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("source path %q is not a regular file or folder", sourceFilePath)
		}

		sourceRelativePath, err := filepath.Rel(sourcePath, sourceFilePath)
		if err != nil {
			return fmt.Errorf("resolve source relative path %q: %w", sourceFilePath, err)
		}

		preview.TotalFileCount++
		preview.TotalSizeBytes += info.Size()
		preview.TargetFilePaths = append(preview.TargetFilePaths, installpath.JoinTargetRelativePath(targetRelativePath, filepath.ToSlash(sourceRelativePath)))
		return nil
	})
}

func buildUnrealPakPreview(sourcePath string, targetRelativePath string, preview *Preview) error {
	inspection, err := unrealpak.Inspect(sourcePath)
	if err != nil {
		return err
	}

	preview.TotalFileCount = len(inspection.Files)
	preview.TotalSizeBytes = inspection.SizeBytes
	preview.Warnings = append(preview.Warnings, inspection.Warnings...)
	for _, file := range inspection.Files {
		preview.TargetFilePaths = append(
			preview.TargetFilePaths,
			installpath.JoinTargetRelativePath(targetRelativePath, file.Name),
		)
	}
	return nil
}
