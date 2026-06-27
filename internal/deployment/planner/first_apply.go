package planner

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/fileops"
)

func PlanFirstApply(state deployment.DesiredState, gameInstallPath string) (plan DeploymentPlan, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("plan first apply: %w", err)
		}
	}()

	plan = DeploymentPlan{
		Mode:   PlanModeFirstApply,
		Paths:  map[string]PathPlan{},
		Issues: append([]deployment.PlanIssue(nil), state.Issues...),
	}

	gameInstallPath = filepath.Clean(gameInstallPath)
	canonicalPaths := sortedCanonicalPaths(state.Files)

	for _, canonicalPath := range canonicalPaths {
		file := state.Files[canonicalPath]
		pathPlan := PathPlan{
			GameRelativePath: file.GameRelativePath,
			FileStatus:       file.FileStatus,
			RiskLevel:        file.RiskLevel,
			ConflictCategory: file.ConflictCategory,
			Desired: FileStateSnapshot{
				Exists:    true,
				SHA256:    file.SHA256,
				SizeBytes: file.SizeBytes,
				Label:     "Desired profile content",
			},
		}

		switch file.FileStatus {
		case deployment.FileStatusBlocked:
			pathPlan.PlannedAction = ReapplyBlock
			pathPlan.Current = FileStateSnapshot{Exists: false}
		case deployment.FileStatusAdded:
			pathPlan.PlannedAction = ReapplyCreate
			pathPlan.Current = FileStateSnapshot{Exists: false}
		case deployment.FileStatusReplaced:
			targetPath := filepath.Join(gameInstallPath, filepath.FromSlash(file.GameRelativePath))
			hash, size, integrityErr := fileops.FileIntegrity(targetPath)
			if integrityErr != nil {
				return DeploymentPlan{}, fmt.Errorf("hash current file %q: %w", file.GameRelativePath, integrityErr)
			}
			pathPlan.PlannedAction = ReapplyReplace
			pathPlan.Current = FileStateSnapshot{
				Exists:    true,
				SHA256:    hash,
				SizeBytes: size,
				Label:     "Current game install",
			}
		default:
			pathPlan.PlannedAction = ReapplyBlock
			pathPlan.Current = FileStateSnapshot{Exists: false}
		}

		plan.Paths[canonicalPath] = pathPlan
	}

	return plan, nil
}

func sortedCanonicalPaths(files map[string]deployment.DesiredFile) []string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
