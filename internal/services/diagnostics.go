package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/phergul/mod-manager/internal/diagnostics"
	"github.com/phergul/mod-manager/internal/services/dto"
	"github.com/phergul/mod-manager/internal/services/dto/mappers"
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

	return mappers.ToDTODiagnosticLogEntries(logs), nil
}

func (s *DiagnosticsService) ListRecentRawLogs(ctx context.Context, input dto.ListDiagnosticLogsInput) (content string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list recent raw diagnostic logs: %w", err)
		}
	}()

	lines, err := s.manager.RecentRawLogs(ctx, diagnostics.RecentLogsInput{
		Limit:     input.Limit,
		Operation: input.Operation,
		Level:     input.Level,
	})
	if err != nil {
		return "", err
	}

	return formatRawDiagnosticLogLines(lines)
}

func (s *DiagnosticsService) ExportLogs(ctx context.Context, input dto.ExportDiagnosticLogsInput) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("export diagnostic logs: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	path := strings.TrimSpace(input.Path)
	if path == "" {
		return errors.New("export path is required")
	}

	content := formatDiagnosticLogEntries(input.Entries)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}

	return nil
}

func formatRawDiagnosticLogLines(lines []string) (string, error) {
	if len(lines) == 0 {
		return "[]", nil
	}

	entries := make([]any, 0, len(lines))
	for _, line := range lines {
		var entry any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		entries = append(entries, entry)
	}

	content, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func formatDiagnosticLogEntries(entries []dto.DiagnosticLogEntry) string {
	if len(entries) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, entry := range entries {
		builder.WriteString(strings.TrimSpace(entry.Timestamp))
		builder.WriteString(" ")
		builder.WriteString(strings.ToUpper(strings.TrimSpace(entry.Level)))

		operation := strings.TrimSpace(entry.Operation)
		if operation != "" {
			builder.WriteString(" [")
			builder.WriteString(operation)
			builder.WriteString("]")
		}

		builder.WriteString(" ")
		builder.WriteString(strings.TrimSpace(entry.Message))
		builder.WriteString("\n")

		if len(entry.Details) > 0 {
			keys := make([]string, 0, len(entry.Details))
			for key := range entry.Details {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			for _, key := range keys {
				value := strings.TrimSpace(entry.Details[key])
				if value == "" {
					continue
				}

				builder.WriteString("  ")
				builder.WriteString(key)
				builder.WriteString(": ")
				builder.WriteString(value)
				builder.WriteString("\n")
			}
		}
	}

	return builder.String()
}
