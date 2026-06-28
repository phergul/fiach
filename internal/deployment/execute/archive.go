package execute

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/fileops"
)

func archiveDriftedFiles(
	gameInstallPath string,
	gameModStoragePath string,
	gameID int64,
	plan planner.DeploymentPlan,
	now time.Time,
) (archiveRoot string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("archive drifted files: %w", err)
		}
	}()

	paths := sortedPlanPaths(plan)
	archivePaths := make([]string, 0)
	for _, canonicalPath := range paths {
		pathPlan := plan.Paths[canonicalPath]
		if !pathPlan.RequiresDriftArchive || !pathPlan.Current.Exists {
			continue
		}
		archivePaths = append(archivePaths, canonicalPath)
	}

	if len(archivePaths) == 0 {
		return "", nil
	}

	archiveRoot = filepath.Join(
		gameModStoragePath,
		"archives",
		"drift",
		fmt.Sprintf("%d", gameID),
		fmt.Sprintf("%d", now.UnixNano()),
	)

	for _, canonicalPath := range archivePaths {
		pathPlan := plan.Paths[canonicalPath]
		sourcePath, sourceErr := targetAbsolutePath(gameInstallPath, pathPlan.GameRelativePath, canonicalPath)
		if sourceErr != nil {
			return "", sourceErr
		}

		relativePath := strings.TrimSpace(pathPlan.GameRelativePath)
		if relativePath == "" {
			relativePath = strings.ReplaceAll(canonicalPath, "\\", "/")
		}

		destinationPath := filepath.Join(archiveRoot, filepath.FromSlash(relativePath))
		if err := copyDriftArchiveFile(sourcePath, destinationPath); err != nil {
			return "", err
		}
	}

	return archiveRoot, nil
}

func copyDriftArchiveFile(sourcePath string, destinationPath string) error {
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return err
	}

	return fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
		SourcePath: sourcePath,
		TargetPath: destinationPath,
		Mode:       0o644,
		OpenLabel:  "deployment drift archive file",
	})
}
