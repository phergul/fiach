package mappers

import (
	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/services/dto"
)

func ToDTODiagnosticLogEntries(entries []diagnostics.LogEntry) []dto.DiagnosticLogEntry {
	if len(entries) == 0 {
		return nil
	}

	result := make([]dto.DiagnosticLogEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, dto.DiagnosticLogEntry{
			Timestamp: entry.Timestamp,
			Level:     entry.Level,
			Operation: entry.Operation,
			Message:   entry.Message,
			Details:   entry.Details,
		})
	}

	return result
}

func ToDTODiagnosticOperations(operations []diagnostics.OperationDescriptor) []dto.DiagnosticOperation {
	if len(operations) == 0 {
		return nil
	}

	result := make([]dto.DiagnosticOperation, 0, len(operations))
	for _, operation := range operations {
		result = append(result, dto.DiagnosticOperation{
			Value: operation.Value,
			Label: operation.Label,
		})
	}

	return result
}

func ToDTODiagnosticOperationGroups(groups []diagnostics.OperationGroup) []dto.DiagnosticOperationGroup {
	if len(groups) == 0 {
		return nil
	}

	result := make([]dto.DiagnosticOperationGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, dto.DiagnosticOperationGroup{
			Area:       group.Area,
			Operations: ToDTODiagnosticOperations(group.Operations),
		})
	}

	return result
}
