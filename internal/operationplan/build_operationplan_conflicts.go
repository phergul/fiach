package operationplan

import (
	"fmt"
	"sort"
	"strings"
)

func appendTargetPathConflictIssues(profileID int64, fileOperations []Operation, issues *[]PlanIssue) {
	targets := make(map[string][]int)
	for index := range fileOperations {
		targets[fileOperations[index].TargetPath] = append(targets[fileOperations[index].TargetPath], index)
	}

	conflictingTargets := make([]string, 0)
	for targetPath, indexes := range targets {
		if len(indexes) > 1 {
			conflictingTargets = append(conflictingTargets, targetPath)
		}
	}
	sort.Strings(conflictingTargets)

	for _, targetPath := range conflictingTargets {
		indexes := targets[targetPath]
		modNames := markConflictingOperations(fileOperations, indexes)

		*issues = append(*issues, newPlanIssue(
			PlanIssueSeverityError,
			PlanIssueTargetPathConflict,
			profileID,
			fmt.Sprintf("multiple planned operations target %q (mods: %s)", targetPath, strings.Join(modNames, ", ")),
			nil,
			nil,
			stringPtr(targetPath),
		))
	}
}

func markConflictingOperations(fileOperations []Operation, indexes []int) []string {
	modNames := make([]string, 0, len(indexes))
	for _, index := range indexes {
		fileOperations[index].Conflict = true
		modNames = append(modNames, fmt.Sprintf("%q", fileOperations[index].Mod.ModName))
	}
	sort.Strings(modNames)
	return modNames
}

func canApplyPlan(issues []PlanIssue) bool {
	for _, issue := range issues {
		if issue.Severity == PlanIssueSeverityError {
			return false
		}
	}

	return true
}
