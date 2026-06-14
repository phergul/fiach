package optiscaler

import (
	"debug/pe"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func DiscoverCandidates(gameRoot string, managedTargetPaths []string) (candidates []Candidate, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("discover OptiScaler targets: %w", err)
		}
	}()

	gameRoot, err = filepath.Abs(strings.TrimSpace(gameRoot))
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(gameRoot)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("game root is not a directory")
	}
	managed := make(map[string]bool, len(managedTargetPaths))
	for _, path := range managedTargetPaths {
		managed[strings.ToLower(filepath.Clean(path))] = true
	}

	err = filepath.WalkDir(gameRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == gameRoot {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".exe") {
			return nil
		}
		if err := requireNoSymlinkComponents(gameRoot, path); err != nil {
			return nil
		}
		isX64, inspectErr := IsX64PE(path)
		if inspectErr != nil || !isX64 {
			return nil
		}

		targetPath := filepath.Dir(path)
		targetRelative, err := RelativeToRoot(gameRoot, targetPath)
		if err != nil {
			return err
		}
		executableRelative, err := RelativeToRoot(gameRoot, path)
		if err != nil {
			return err
		}
		evidence := candidateEvidence(targetRelative, entry.Name(), managed[strings.ToLower(targetRelative)])
		hasOptiScaler, hasReShade := inspectTargetMarkers(targetPath)
		if hasOptiScaler {
			evidence = append(evidence, "OptiScaler-owned runtime detected")
		}
		if hasReShade {
			evidence = append(evidence, "ReShade-owned runtime detected")
		}
		candidates = append(candidates, Candidate{
			TargetRelativePath:     targetRelative,
			ExecutableRelativePath: executableRelative,
			ExecutableName:         entry.Name(),
			Architecture:           "x64",
			Evidence:               evidence,
			Managed:                managed[strings.ToLower(targetRelative)],
			HasOptiScaler:          hasOptiScaler,
			HasReShade:             hasReShade,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidateScore(candidates[i]), candidateScore(candidates[j])
		if left != right {
			return left > right
		}
		leftDepth := pathDepth(candidates[i].TargetRelativePath)
		rightDepth := pathDepth(candidates[j].TargetRelativePath)
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return strings.ToLower(candidates[i].ExecutableRelativePath) < strings.ToLower(candidates[j].ExecutableRelativePath)
	})
	if candidates == nil {
		candidates = []Candidate{}
	}
	return candidates, nil
}

func IsX64PE(path string) (isX64 bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("inspect executable architecture: %w", err)
		}
	}()
	file, err := pe.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	return file.FileHeader.Machine == pe.IMAGE_FILE_MACHINE_AMD64, nil
}

func candidateEvidence(targetRelative string, executableName string, managed bool) []string {
	evidence := []string{"validated Windows x64 executable"}
	lowerTarget := strings.ToLower(filepath.ToSlash(targetRelative))
	lowerName := strings.ToLower(executableName)
	if managed {
		evidence = append(evidence, "already managed by Fiach")
	}
	if strings.Contains(lowerTarget, "/binaries/win64") || strings.HasSuffix(lowerTarget, "binaries/win64") {
		evidence = append(evidence, "common Unreal Win64 directory")
	}
	if strings.Contains(lowerTarget, "/wingdk") || strings.HasSuffix(lowerTarget, "wingdk") {
		evidence = append(evidence, "common WinGDK directory")
	}
	if strings.Contains(lowerName, "-win64-shipping") || strings.Contains(lowerName, "shipping") {
		evidence = append(evidence, "shipping executable name")
	}
	return evidence
}

func candidateScore(candidate Candidate) int {
	score := 0
	if candidate.Managed {
		score += 1000
	}
	if candidate.HasOptiScaler {
		score += 500
	}
	if candidate.HasReShade {
		score += 250
	}
	for _, evidence := range candidate.Evidence {
		switch evidence {
		case "common Unreal Win64 directory", "common WinGDK directory":
			score += 100
		case "shipping executable name":
			score += 50
		}
	}
	return score
}

func pathDepth(path string) int {
	if path == "." {
		return 0
	}
	return len(strings.Split(filepath.Clean(path), string(filepath.Separator)))
}

func inspectTargetMarkers(targetPath string) (bool, bool) {
	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return false, false
	}
	hasOptiScaler, hasReShade := false, false
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".dll") {
			continue
		}
		owner, err := InspectOwnership(filepath.Join(targetPath, entry.Name()))
		if err != nil {
			continue
		}
		hasOptiScaler = hasOptiScaler || owner == OwnershipOptiScaler
		hasReShade = hasReShade || owner == OwnershipReShade
	}
	return hasOptiScaler, hasReShade
}
