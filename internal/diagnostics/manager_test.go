package diagnostics

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManagerRotatesLogsAndReadsNewestEntries(t *testing.T) {
	t.Parallel()

	manager, err := NewManager(Options{
		LogPath:     filepath.Join(t.TempDir(), DefaultLogFileName),
		MaxFileSize: 700,
		MaxFiles:    3,
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer closeManager(t, manager)

	for index := 0; index < 12; index++ {
		manager.Logger().Info("Game scan completed",
			slog.String("operation", OperationScanGames),
			slog.String("event", EventCompleted),
			slog.Int("inserted_count", index),
		)
	}

	entries, err := manager.RecentLogs(context.Background(), RecentLogsInput{Limit: 5})
	if err != nil {
		t.Fatalf("RecentLogs() error = %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("RecentLogs() length = %d, want 5", len(entries))
	}
	if entries[0].Details["Inserted count"] != "11" {
		t.Fatalf("newest entry details = %+v, want inserted count 11", entries[0].Details)
	}

	files, err := filepath.Glob(filepath.Join(filepath.Dir(manager.writer.path), DefaultLogFileName+"*"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(files) > 3 {
		t.Fatalf("rotated log files = %v, want at most 3", files)
	}
}

func TestRecentLogsSkipsMalformedLinesAndCleansDetails(t *testing.T) {
	t.Parallel()

	logPath := filepath.Join(t.TempDir(), DefaultLogFileName)
	if err := os.WriteFile(logPath, []byte(`not-json
{"time":"2026-05-27T12:00:00Z","level":"INFO","msg":"Mod import completed","operation":"import_mod","event":"completed","source_path":"/Users/fergal/Games/Mod.zip","source_path_label":"Games/Mod.zip","game_id":42}
{"time":"2026-05-27T12:01:00Z","level":"ERROR","msg":"Mod import failed","operation":"import_mod","event":"failed","error":"copy source \"/Users/fergal/Games/Mod.zip\": remove /Users/fergal/Games/Mod.zip: permission denied"}
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manager, err := NewManager(Options{LogPath: logPath})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer closeManager(t, manager)

	entries, err := manager.RecentLogs(context.Background(), RecentLogsInput{Limit: 10})
	if err != nil {
		t.Fatalf("RecentLogs() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("RecentLogs() length = %d, want 2", len(entries))
	}
	wantError := `copy source "` + filepath.Join("Games", "Mod.zip") + `": remove ` + filepath.Join("Games", "Mod.zip") + `: permission denied`
	if entries[0].Details["Error"] != wantError {
		t.Fatalf("RecentLogs() newest details = %+v, want sanitized error paths", entries[0].Details)
	}
	if _, ok := entries[1].Details["Source path"]; ok {
		t.Fatalf("RecentLogs() details = %+v, want raw path omitted", entries[1].Details)
	}
	if entries[1].Details["Source path label"] != "Games/Mod.zip" {
		t.Fatalf("RecentLogs() details = %+v, want safe path label", entries[1].Details)
	}
	if entries[1].Details["Game id"] != "42" {
		t.Fatalf("RecentLogs() details = %+v, want game id", entries[1].Details)
	}
}

func TestRecentRawLogsReturnsFilteredRawLines(t *testing.T) {
	t.Parallel()

	logPath := filepath.Join(t.TempDir(), DefaultLogFileName)
	if err := os.WriteFile(logPath, []byte(`not-json
{"time":"2026-05-27T12:00:00Z","level":"INFO","msg":"Mod import completed","operation":"import_mod","event":"completed","source_path":"/Users/fergal/Games/Mod.zip","game_id":42}
{"time":"2026-05-27T12:01:00Z","level":"ERROR","msg":"Mod import failed","operation":"import_mod","event":"failed","source_path":"/Users/fergal/Games/Broken.zip"}
{"time":"2026-05-27T12:02:00Z","level":"INFO","msg":"Game scan completed","operation":"scan_games","event":"completed","inserted_count":1}
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manager, err := NewManager(Options{LogPath: logPath})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer closeManager(t, manager)

	lines, err := manager.RecentRawLogs(context.Background(), RecentLogsInput{
		Limit:     1,
		Level:     "info",
		Operation: OperationImportMod,
	})
	if err != nil {
		t.Fatalf("RecentRawLogs() error = %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("RecentRawLogs() length = %d, want 1", len(lines))
	}
	if !strings.Contains(lines[0], `"source_path":"/Users/fergal/Games/Mod.zip"`) {
		t.Fatalf("RecentRawLogs() line = %q, want raw source path preserved", lines[0])
	}
}

func TestManagerBroadcastsSanitizedWrittenLogs(t *testing.T) {
	t.Parallel()

	manager, err := NewManager(Options{
		LogPath: filepath.Join(t.TempDir(), DefaultLogFileName),
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer closeManager(t, manager)

	entries, unsubscribe := manager.Subscribe()
	defer unsubscribe()

	manager.Logger().Error("Mod import failed",
		slog.String("operation", OperationImportMod),
		slog.String("event", EventFailed),
		slog.String("error", `copy source "/Users/fergal/Games/Mod.zip": permission denied`),
	)

	select {
	case entry := <-entries:
		if entry.Level != "error" || entry.Operation != OperationImportMod {
			t.Fatalf("broadcast entry = %+v, want error import entry", entry)
		}
		wantError := `copy source "` + filepath.Join("Games", "Mod.zip") + `": permission denied`
		if entry.Details["Error"] != wantError {
			t.Fatalf("broadcast details = %+v, want sanitized error path", entry.Details)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast log entry")
	}
}

func TestPathLabelUsesOnlyLastPathSegments(t *testing.T) {
	t.Parallel()

	got := PathLabel(filepath.Join("Users", "fergal", "Games", "Mod.zip"))
	if got != filepath.Join("Games", "Mod.zip") {
		t.Fatalf("PathLabel() = %q, want Games/Mod.zip", got)
	}
}

func closeManager(t *testing.T, manager *Manager) {
	t.Helper()

	if err := manager.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
