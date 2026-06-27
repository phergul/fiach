package review

import (
	"path"
	"sort"
	"strings"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
)

type TreeNode struct {
	Path          string
	Name          string
	IsDirectory   bool
	Status        deployment.FileStatus
	PlannedAction planner.ReapplyAction
	RiskLevel     deployment.RiskLevel
	ChildCount    int
	HasChildren   bool
}

type childAccumulator struct {
	name          string
	displayPath   string
	isDirectory   bool
	status        deployment.FileStatus
	plannedAction planner.ReapplyAction
	riskLevel     deployment.RiskLevel
	childCount    int
}

func BuildTreeChildren(plan planner.FirstApplyPlan, parentPath string) []TreeNode {
	children := map[string]*childAccumulator{}
	parentSlash := normalizeTreePath(parentPath)

	for _, pathPlan := range plan.Paths {
		childName, childPath, isFile, ok := directChild(parentSlash, pathPlan.GameRelativePath)
		if !ok {
			continue
		}

		key := strings.ToLower(childName)
		existing, found := children[key]
		if !found {
			accumulator := &childAccumulator{
				name:          childName,
				displayPath:   childPath,
				isDirectory:   !isFile,
				status:        pathPlan.FileStatus,
				plannedAction: pathPlan.PlannedAction,
				riskLevel:     pathPlan.RiskLevel,
				childCount:    0,
			}
			if isFile {
				accumulator.isDirectory = false
			}
			children[key] = accumulator
			continue
		}

		if isFile {
			existing.isDirectory = false
			existing.displayPath = childPath
			existing.status = pathPlan.FileStatus
			existing.plannedAction = pathPlan.PlannedAction
			existing.riskLevel = pathPlan.RiskLevel
		} else {
			existing.isDirectory = true
		}

		existing.status = RollUpStatus(existing.status, pathPlan.FileStatus)
		if riskLevelPriority(pathPlan.RiskLevel) > riskLevelPriority(existing.riskLevel) {
			existing.riskLevel = pathPlan.RiskLevel
		}
		if plannedActionPriority(pathPlan.PlannedAction) > plannedActionPriority(existing.plannedAction) {
			existing.plannedAction = pathPlan.PlannedAction
		}
	}

	for key, accumulator := range children {
		if accumulator.isDirectory {
			accumulator.childCount = countDirectChildren(plan, accumulator.displayPath)
		}
		children[key] = accumulator
	}

	keys := make([]string, 0, len(children))
	for key := range children {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a := children[keys[i]]
		b := children[keys[j]]
		if a.isDirectory != b.isDirectory {
			return a.isDirectory
		}
		return strings.ToLower(a.name) < strings.ToLower(b.name)
	})

	nodes := make([]TreeNode, 0, len(keys))
	for _, key := range keys {
		accumulator := children[key]
		nodes = append(nodes, TreeNode{
			Path:          accumulator.displayPath,
			Name:          accumulator.name,
			IsDirectory:   accumulator.isDirectory,
			Status:        accumulator.status,
			PlannedAction: accumulator.plannedAction,
			RiskLevel:     accumulator.riskLevel,
			ChildCount:    accumulator.childCount,
			HasChildren:   accumulator.isDirectory && accumulator.childCount > 0,
		})
	}

	return nodes
}

func countDirectChildren(plan planner.FirstApplyPlan, parentPath string) int {
	seen := map[string]struct{}{}
	parentSlash := normalizeTreePath(parentPath)

	for _, pathPlan := range plan.Paths {
		childName, _, _, ok := directChild(parentSlash, pathPlan.GameRelativePath)
		if !ok {
			continue
		}
		seen[strings.ToLower(childName)] = struct{}{}
	}

	return len(seen)
}

func directChild(parentPath string, gameRelativePath string) (childName string, childPath string, isFile bool, ok bool) {
	normalized := normalizeTreePath(gameRelativePath)
	if normalized == "" {
		return "", "", false, false
	}

	if parentPath == "" {
		if slashIndex := strings.Index(normalized, "/"); slashIndex >= 0 {
			childName = normalized[:slashIndex]
			return childName, childName, false, true
		}
		return normalized, normalized, true, true
	}

	prefix := parentPath + "/"
	if !strings.HasPrefix(strings.ToLower(normalized), strings.ToLower(prefix)) {
		return "", "", false, false
	}

	remainder := normalized[len(prefix):]
	if remainder == "" {
		return "", "", false, false
	}

	if slashIndex := strings.Index(remainder, "/"); slashIndex >= 0 {
		childName = remainder[:slashIndex]
		return childName, parentPath + "/" + childName, false, true
	}

	return remainder, parentPath + "/" + remainder, true, true
}

func normalizeTreePath(value string) string {
	cleaned := strings.TrimPrefix(path.Clean(strings.ReplaceAll(value, "\\", "/")), "/")
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func riskLevelPriority(level deployment.RiskLevel) int {
	switch level {
	case deployment.RiskError:
		return 3
	case deployment.RiskInfo:
		return 2
	default:
		return 1
	}
}

func plannedActionPriority(action planner.ReapplyAction) int {
	switch action {
	case planner.ReapplyBlock:
		return 3
	case planner.ReapplyReplace:
		return 2
	case planner.ReapplyCreate:
		return 1
	default:
		return 0
	}
}
