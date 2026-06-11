package unrealpak

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

const modsDirectoryName = "~mods"

type TargetDetection struct {
	Candidates []string
	Warnings   []string
}

func DetectTargets(gameInstallPath string) (result TargetDetection, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("detect Unreal package targets: %w", err)
		}
	}()

	result.Candidates = []string{}
	result.Warnings = []string{}
	seen := map[string]struct{}{}

	err = filepath.WalkDir(gameInstallPath, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if currentPath == gameInstallPath {
				return walkErr
			}
			result.Warnings = append(
				result.Warnings,
				fmt.Sprintf("Could not inspect %q while detecting Unreal package targets.", currentPath),
			)
			return filepath.SkipDir
		}
		if currentPath == gameInstallPath || !entry.IsDir() {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return filepath.SkipDir
		}
		if !strings.EqualFold(entry.Name(), "Paks") ||
			!strings.EqualFold(filepath.Base(filepath.Dir(currentPath)), "Content") {
			return nil
		}

		relativePath, relErr := filepath.Rel(gameInstallPath, currentPath)
		if relErr != nil {
			return fmt.Errorf("resolve package target %q: %w", currentPath, relErr)
		}
		candidate := filepath.ToSlash(filepath.Join(relativePath, modsDirectoryName))
		key := strings.ToLower(candidate)
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			result.Candidates = append(result.Candidates, candidate)
		}
		return filepath.SkipDir
	})
	if err != nil {
		return TargetDetection{}, err
	}

	sort.SliceStable(result.Candidates, func(i int, j int) bool {
		return strings.ToLower(result.Candidates[i]) < strings.ToLower(result.Candidates[j])
	})
	if len(result.Candidates) == 0 {
		result.Warnings = append(
			result.Warnings,
			"No existing Content/Paks folder was found. Enter a game-relative target path manually.",
		)
	}

	return result, nil
}

func TargetWasDetected(targetRelativePath string, candidates []string) bool {
	targetKey := strings.ToLower(filepath.ToSlash(filepath.Clean(targetRelativePath)))
	for _, candidate := range candidates {
		if targetKey == strings.ToLower(filepath.ToSlash(filepath.Clean(candidate))) {
			return true
		}
	}
	return false
}
