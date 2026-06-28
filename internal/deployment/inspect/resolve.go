package inspect

import (
	"fmt"
	"os"
	"strings"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/deployment/review"
	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/filetxn"
)

type resolvedStatePath struct {
	Path      string
	Available bool
	Reason    string
	SHA256    string
	SizeBytes int64
}

func resolveStatePaths(
	entry review.CachedPreview,
	gameInstallPath string,
	relativePath string,
) (map[StateKind]resolvedStatePath, error) {
	canonicalPath := deployment.CanonicalGameRelativePath(relativePath)
	pathPlan, found := entry.Plan.Paths[canonicalPath]
	if !found {
		return nil, fmt.Errorf("deployment path %q was not found in preview", relativePath)
	}

	desiredFile, hasDesired := entry.Desired.Files[canonicalPath]

	result := map[StateKind]resolvedStatePath{
		StateBaseline: {Available: false, Reason: "Baseline content is not available for inspection."},
		StateApplied:  {Available: false, Reason: "Last applied content is not available for inspection."},
		StateCurrent:  {Available: false, Reason: "Current file is not present on disk."},
		StateDesired:  {Available: false, Reason: "Desired content is not available for inspection."},
	}

	if pathPlan.BaselineBackupPath != "" {
		resolved, err := resolveExistingFile(pathPlan.BaselineBackupPath, pathPlan.Baseline)
		if err != nil {
			return nil, err
		}
		result[StateBaseline] = resolved
	}

	if gameInstallPath != "" && pathPlan.GameRelativePath != "" {
		currentPath, err := filetxn.ResolveWithinRoot(gameInstallPath, pathPlan.GameRelativePath)
		if err != nil {
			return nil, fmt.Errorf("resolve current file path: %w", err)
		}
		resolved, err := resolveExistingFile(currentPath, pathPlan.Current)
		if err != nil {
			return nil, err
		}
		result[StateCurrent] = resolved
	}

	if hasDesired && strings.TrimSpace(desiredFile.SourcePath) != "" {
		resolved, err := resolveExistingFile(desiredFile.SourcePath, pathPlan.Desired)
		if err != nil {
			return nil, err
		}
		result[StateDesired] = resolved
	}

	result[StateApplied] = resolveAppliedPath(pathPlan.Applied, result)

	return result, nil
}

func resolveAppliedPath(
	applied planner.FileStateSnapshot,
	states map[StateKind]resolvedStatePath,
) resolvedStatePath {
	if !applied.Exists || applied.SHA256 == "" {
		return resolvedStatePath{
			Available: false,
			Reason:    "Last applied content is not available for inspection.",
		}
	}

	candidates := []StateKind{StateCurrent, StateDesired, StateBaseline}
	for _, kind := range candidates {
		candidate := states[kind]
		if !candidate.Available {
			continue
		}
		if strings.EqualFold(candidate.SHA256, applied.SHA256) {
			return candidate
		}
	}

	return resolvedStatePath{
		Available: false,
		Reason:    "Last applied content is no longer available at a known location.",
	}
}

func resolveExistingFile(path string, snapshot planner.FileStateSnapshot) (resolvedStatePath, error) {
	_, err := fileops.StatRegularFile("inspection file", path)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "does not exist") {
			return resolvedStatePath{
				Available: false,
				Reason:    "File is not present.",
			}, nil
		}
		return resolvedStatePath{}, err
	}

	sha256Hex := snapshot.SHA256
	sizeBytes := snapshot.SizeBytes
	if sha256Hex == "" || sizeBytes == 0 {
		sha256Hex, sizeBytes, err = fileops.FileIntegrity(path)
		if err != nil {
			return resolvedStatePath{}, fmt.Errorf("hash inspection file %q: %w", path, err)
		}
	}

	return resolvedStatePath{
		Path:      path,
		Available: true,
		SHA256:    sha256Hex,
		SizeBytes: sizeBytes,
	}, nil
}

func toSideMetadata(kind StateKind, resolved resolvedStatePath) SideMetadata {
	return SideMetadata{
		StateKind:         kind,
		Label:             stateLabel(kind),
		Available:         resolved.Available,
		UnavailableReason: resolved.Reason,
		SHA256:            resolved.SHA256,
		SizeBytes:         resolved.SizeBytes,
	}
}
