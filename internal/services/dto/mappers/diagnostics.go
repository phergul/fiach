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
