package installconfig

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/phergul/mod-manager/internal/installpath"
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
	if input.StrategyType != StrategyTypeGenericCopy {
		return Preview{}, fmt.Errorf("import strategy %q cannot be previewed", input.StrategyType)
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

	targetPaths := make([]string, 0)
	err = filepath.WalkDir(input.SourcePath, func(sourceFilePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if sourceFilePath == input.SourcePath {
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

		sourceRelativePath, err := filepath.Rel(input.SourcePath, sourceFilePath)
		if err != nil {
			return fmt.Errorf("resolve source relative path %q: %w", sourceFilePath, err)
		}

		preview.TotalFileCount++
		targetPaths = append(targetPaths, installpath.JoinTargetRelativePath(targetRelativePath, filepath.ToSlash(sourceRelativePath)))
		return nil
	})
	if err != nil {
		return Preview{}, err
	}

	sort.Strings(targetPaths)
	if len(targetPaths) > fileCap {
		preview.IsCapped = true
		preview.TargetFilePaths = targetPaths[:fileCap]
		preview.Warnings = append(preview.Warnings, fmt.Sprintf("Showing first %d of %d target files.", fileCap, len(targetPaths)))
	} else {
		preview.TargetFilePaths = targetPaths
	}

	return preview, nil
}
