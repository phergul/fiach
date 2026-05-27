package services

import (
	"context"
	"fmt"

	"github.com/phergul/mod-manager/internal/diagnostics"
	"github.com/phergul/mod-manager/internal/services/dto"
)

type DiagnosticsService struct {
	manager *diagnostics.Manager
}

func NewDiagnosticsService(manager *diagnostics.Manager) *DiagnosticsService {
	return &DiagnosticsService{
		manager: manager,
	}
}

func (s *DiagnosticsService) ListRecentLogs(ctx context.Context, input dto.ListDiagnosticLogsInput) (entries []dto.DiagnosticLogEntry, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list recent diagnostic logs: %w", err)
		}
	}()

	logs, err := s.manager.RecentLogs(ctx, diagnostics.RecentLogsInput{
		Limit:     input.Limit,
		Operation: input.Operation,
		Level:     input.Level,
	})
	if err != nil {
		return nil, err
	}

	return toDTODiagnosticLogEntries(logs), nil
}

func toDTODiagnosticLogEntries(entries []diagnostics.LogEntry) []dto.DiagnosticLogEntry {
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
