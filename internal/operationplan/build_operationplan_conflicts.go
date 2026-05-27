package operationplan

import (
	"fmt"
	"sort"
)

func appendTargetPathConflictIssues(profileID int64, fileOperationOffset int, fileOperations []Operation, issues *[]PlanIssue) {
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
		markConflictingOperations(fileOperations, indexes)
		conflictingOperationIndexes := finalOperationIndexes(fileOperationOffset, indexes)

		issue := newPlanIssue(
			PlanIssueSeverityError,
			PlanIssueTargetPathConflict,
			profileID,
			fmt.Sprintf("multiple planned operations target %q", targetPath),
			nil,
			nil,
			stringPtr(targetPath),
		)
		issue.ConflictingOperationIndexes = conflictingOperationIndexes
		*issues = append(*issues, issue)
	}
}

func markConflictingOperations(fileOperations []Operation, indexes []int) {
	for _, index := range indexes {
		fileOperations[index].Conflict = true
	}
}

func finalOperationIndexes(fileOperationOffset int, fileOperationIndexes []int) []int {
	indexes := make([]int, 0, len(fileOperationIndexes))
	for _, index := range fileOperationIndexes {
		indexes = append(indexes, fileOperationOffset+index)
	}
	sort.Ints(indexes)
	return indexes
}

func canApplyPlan(issues []PlanIssue) bool {
	for _, issue := range issues {
		if issue.Severity == PlanIssueSeverityError {
			return false
		}
	}

	return true
}
