package applyplan

import "github.com/phergul/mod-manager/internal/operationplan"

func newFailedResult(index int, operation operationplan.Operation, err error) operationplan.ApplyOperationResult {
	errorMessage := err.Error()
	return operationplan.ApplyOperationResult{
		OperationIndex: index,
		Operation:      operation,
		Status:         operationplan.ApplyOperationStatusFailed,
		Message:        "Operation failed.",
		Error:          &errorMessage,
	}
}

func appendSkippedResults(operations []operationplan.Operation, startIndex int, result *operationplan.ApplyOperationPlanResult) {
	for index := startIndex; index < len(operations); index++ {
		result.Results = append(result.Results, operationplan.ApplyOperationResult{
			OperationIndex: index,
			Operation:      operations[index],
			Status:         operationplan.ApplyOperationStatusSkipped,
			Message:        "Skipped after a previous operation failed.",
		})
	}
}

func updateCounts(result *operationplan.ApplyOperationPlanResult) {
	result.CompletedCount = 0
	result.FailedCount = 0
	result.SkippedCount = 0

	for _, operationResult := range result.Results {
		switch operationResult.Status {
		case operationplan.ApplyOperationStatusCompleted:
			result.CompletedCount++
		case operationplan.ApplyOperationStatusFailed:
			result.FailedCount++
		case operationplan.ApplyOperationStatusSkipped:
			result.SkippedCount++
		}
	}
}
