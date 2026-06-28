package drift

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/fileops"
)

func DetectForPaths(
	installPath string,
	applied []appliedstate.PersistedFileState,
	paths []string,
) (results []Result, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("detect drift for paths: %w", err)
		}
	}()

	appliedByPath := map[string]appliedstate.PersistedFileState{}
	for _, state := range applied {
		key := deployment.CanonicalGameRelativePath(state.GameRelativePath)
		appliedByPath[key] = state
	}

	results = make([]Result, 0, len(paths))
	installPath = filepath.Clean(installPath)

	for _, path := range paths {
		canonicalPath := deployment.CanonicalGameRelativePath(path)
		state, found := appliedByPath[canonicalPath]
		if !found {
			continue
		}

		result, detectErr := detectPath(installPath, state)
		if detectErr != nil {
			return nil, detectErr
		}
		results = append(results, result)
	}

	return results, nil
}

func DetectAll(installPath string, applied []appliedstate.PersistedFileState) (results []Result, err error) {
	paths := make([]string, len(applied))
	for index, state := range applied {
		paths[index] = state.GameRelativePath
	}

	return DetectForPaths(installPath, applied, paths)
}

func detectPath(installPath string, state appliedstate.PersistedFileState) (Result, error) {
	result := Result{
		GameRelativePath: state.GameRelativePath,
		AppliedExists:    state.AppliedExists,
	}

	if state.AppliedSHA256 != nil {
		result.AppliedSHA256 = *state.AppliedSHA256
	}
	if state.AppliedSizeBytes != nil {
		result.AppliedSizeBytes = *state.AppliedSizeBytes
	}

	if IsSkippedDecision(state.UserDecision) {
		current, currentErr := readCurrentState(installPath, state.GameRelativePath)
		if currentErr != nil {
			return Result{}, currentErr
		}
		result.CurrentExists = current.Exists
		result.CurrentSHA256 = current.SHA256
		result.CurrentSizeBytes = current.SizeBytes
		result.Kind = deployment.DriftNone
		return result, nil
	}

	if IsKeepExternalDecision(state.UserDecision) {
		result.Kind = deployment.DriftExternal
		current, currentErr := readCurrentState(installPath, state.GameRelativePath)
		if currentErr != nil {
			return Result{}, currentErr
		}
		result.CurrentExists = current.Exists
		result.CurrentSHA256 = current.SHA256
		result.CurrentSizeBytes = current.SizeBytes
		return result, nil
	}

	if !state.AppliedExists {
		result.Kind = deployment.DriftNone
		return result, nil
	}

	current, err := readCurrentState(installPath, state.GameRelativePath)
	if err != nil {
		return Result{}, err
	}

	result.CurrentExists = current.Exists
	result.CurrentSHA256 = current.SHA256
	result.CurrentSizeBytes = current.SizeBytes

	if !current.Exists {
		result.Kind = deployment.DriftMissing
		return result, nil
	}

	if !matchesAppliedIntegrity(result.AppliedSHA256, result.AppliedSizeBytes, current.SHA256, current.SizeBytes) {
		result.Kind = deployment.DriftModified
		return result, nil
	}

	result.Kind = deployment.DriftNone
	return result, nil
}

type currentState struct {
	Exists    bool
	SHA256    string
	SizeBytes int64
}

func readCurrentState(installPath string, gameRelativePath string) (currentState, error) {
	targetPath := filepath.Join(installPath, filepath.FromSlash(gameRelativePath))
	hash, size, err := fileops.FileIntegrity(targetPath)
	if errors.Is(err, os.ErrNotExist) {
		return currentState{Exists: false}, nil
	}
	if err != nil {
		return currentState{}, fmt.Errorf("hash current file %q: %w", gameRelativePath, err)
	}

	return currentState{
		Exists:    true,
		SHA256:    hash,
		SizeBytes: size,
	}, nil
}

func matchesAppliedIntegrity(appliedSHA256 string, appliedSizeBytes int64, currentSHA256 string, currentSizeBytes int64) bool {
	if appliedSHA256 == "" {
		return false
	}

	return strings.EqualFold(appliedSHA256, currentSHA256) && appliedSizeBytes == currentSizeBytes
}
