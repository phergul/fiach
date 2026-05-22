package restoreplan

func failedPreflightResult(operations []RestoreOperation, failures map[int]error) RestoreResult {
	result := RestoreResult{
		Success: false,
		Results: make([]RestoreOperationResult, 0, len(operations)),
	}
	for index, operation := range operations {
		if err, failed := failures[index]; failed {
			result.Results = append(result.Results, newFailedResult(index, operation, err))
			continue
		}

		result.Results = append(result.Results, RestoreOperationResult{
			OperationIndex: index,
			Operation:      operation,
			Status:         RestoreOperationStatusSkipped,
			Message:        "Skipped because restore preflight failed.",
		})
	}
	updateCounts(&result)

	return result
}

func newFailedResult(index int, operation RestoreOperation, err error) RestoreOperationResult {
	errorMessage := err.Error()
	return RestoreOperationResult{
		OperationIndex: index,
		Operation:      operation,
		Status:         RestoreOperationStatusFailed,
		Message:        "Restore operation failed.",
		Error:          &errorMessage,
	}
}

func appendSkippedResults(operations []RestoreOperation, startIndex int, result *RestoreResult) {
	for index := startIndex; index < len(operations); index++ {
		result.Results = append(result.Results, RestoreOperationResult{
			OperationIndex: index,
			Operation:      operations[index],
			Status:         RestoreOperationStatusSkipped,
			Message:        "Skipped after a previous restore operation failed.",
		})
	}
}

func updateCounts(result *RestoreResult) {
	result.CompletedCount = 0
	result.FailedCount = 0
	result.SkippedCount = 0

	for _, operationResult := range result.Results {
		switch operationResult.Status {
		case RestoreOperationStatusCompleted:
			result.CompletedCount++
		case RestoreOperationStatusFailed:
			result.FailedCount++
		case RestoreOperationStatusSkipped:
			result.SkippedCount++
		}
	}
}
