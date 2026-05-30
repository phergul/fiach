package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/gamesource"
	"github.com/phergul/fiach/internal/services/dto"
)

func TestGamesServiceScanWritesDiagnostics(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)
	manager := newServiceTestDiagnostics(t)
	defer closeServiceTestDiagnostics(t, manager)

	steamRoot := createSteamRoot(t)
	writeLibraryFoldersVDF(t, steamRoot, `
"libraryfolders"
{
	"0"
	{
		"path"		"`+steamRoot+`"
	}
}
`)
	writeAppManifest(t, steamRoot, "appmanifest_1.acf", validManifest("1", "Game One", "GameOne"))
	if err := store.SetSetting(context.Background(), gamesource.SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	service := NewGamesService(store, manager.Logger(), gamesource.NewSteamSource(store))
	if _, err := service.ScanAndSaveGames(context.Background()); err != nil {
		t.Fatalf("ScanAndSaveGames() error = %v", err)
	}

	entries := readServiceTestLogs(t, manager)
	if !hasDiagnosticEvent(entries, diagnostics.OperationScanGames, diagnostics.EventStarted) {
		t.Fatalf("diagnostic entries = %+v, want scan started event", entries)
	}
	completed, ok := findDiagnosticEvent(entries, diagnostics.OperationScanGames, diagnostics.EventCompleted)
	if !ok {
		t.Fatalf("diagnostic entries = %+v, want scan completed event", entries)
	}
	if completed.Details["Inserted count"] != "1" {
		t.Fatalf("completed scan details = %+v, want inserted count 1", completed.Details)
	}
}

func TestModServiceImportWritesDiagnostics(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)
	manager := newServiceTestDiagnostics(t)
	defer closeServiceTestDiagnostics(t, manager)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", t.TempDir())
	sourcePath := makeSourceFolder(t, map[string]string{
		"Data/SkyUI.esp": "plugin",
	})

	service := NewModService(store, manager.Logger())
	result, err := service.ImportMod(context.Background(), dto.ImportModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
		TargetRelativePath: ".",
	})
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	completed, ok := findDiagnosticEvent(readServiceTestLogs(t, manager), diagnostics.OperationImportMod, diagnostics.EventCompleted)
	if !ok {
		t.Fatal("ImportMod() did not write completed diagnostic event")
	}
	if completed.Details["Game id"] != int64String(gameID) || completed.Details["Mod id"] != int64String(result.Mod.ID) {
		t.Fatalf("completed import details = %+v, want game and mod IDs", completed.Details)
	}
	if _, ok := completed.Details["Source path"]; ok {
		t.Fatalf("completed import details = %+v, want raw source path omitted", completed.Details)
	}
	if completed.Details["Source path label"] == "" {
		t.Fatalf("completed import details = %+v, want safe source path label", completed.Details)
	}
}

func TestProfileServiceApplyAndRestoreWriteDiagnostics(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)
	manager := newServiceTestDiagnostics(t)
	defer closeServiceTestDiagnostics(t, manager)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	modID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", "/managed/skyui")
	addServiceProfileMod(t, store, profileID, modID, true, 0)

	sourceRoot := makeProfilePlanSourceTree(t, map[string]string{
		"Data/modded.txt": "modded",
	})
	sourceFilePath := filepath.Join(sourceRoot, "Data", "modded.txt")
	targetPath := filepath.Join(gameRoot, "Data", "modded.txt")
	backupPath := filepath.Join(serviceRestoreGameModStoragePath(t, store, gameID), "operation-backups", "Data", "modded.txt")

	service := NewProfileService(store, manager.Logger())
	applyResult, err := service.ApplyProfileOperationPlan(context.Background(), profileID, dto.OperationPlan{
		CanApply: true,
		Operations: []dto.Operation{
			{
				Type:       dto.OperationTypeCreateDirectory,
				TargetPath: filepath.Dir(targetPath),
			},
			{
				Type:       dto.OperationTypeCopy,
				SourcePath: &sourceFilePath,
				TargetPath: targetPath,
				BackupPath: &backupPath,
				Mod: dto.ModContext{
					ModID:   modID,
					ModName: "SkyUI",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyProfileOperationPlan() error = %v", err)
	}
	if !applyResult.Success {
		t.Fatalf("ApplyProfileOperationPlan() = %+v, want success", applyResult)
	}

	restoreResult, err := service.RestoreVanillaState(context.Background(), gameID)
	if err != nil {
		t.Fatalf("RestoreVanillaState() error = %v", err)
	}
	if !restoreResult.Success {
		t.Fatalf("RestoreVanillaState() = %+v, want success", restoreResult)
	}

	entries := readServiceTestLogs(t, manager)
	applyCompleted, ok := findDiagnosticEvent(entries, diagnostics.OperationApplyProfile, diagnostics.EventCompleted)
	if !ok {
		t.Fatalf("diagnostic entries = %+v, want apply completed event", entries)
	}
	if applyCompleted.Details["Profile id"] != int64String(profileID) || applyCompleted.Details["Game id"] != int64String(gameID) {
		t.Fatalf("apply completed details = %+v, want game/profile IDs", applyCompleted.Details)
	}

	restoreCompleted, ok := findDiagnosticEvent(entries, diagnostics.OperationRestoreVanilla, diagnostics.EventCompleted)
	if !ok {
		t.Fatalf("diagnostic entries = %+v, want restore completed event", entries)
	}
	if restoreCompleted.Details["Profile id"] != int64String(profileID) || restoreCompleted.Details["Game id"] != int64String(gameID) {
		t.Fatalf("restore completed details = %+v, want game/profile IDs", restoreCompleted.Details)
	}
}

func TestDiagnosticsServiceListRecentRawLogsReturnsPrettyJSON(t *testing.T) {
	t.Parallel()

	logPath := filepath.Join(t.TempDir(), diagnostics.DefaultLogFileName)
	if err := os.WriteFile(logPath, []byte(`not-json
{"time":"2026-05-27T12:00:00Z","level":"INFO","msg":"Mod import completed","operation":"import_mod","event":"completed","source_path":"/Users/fergal/Games/Mod.zip","game_id":42}
{"time":"2026-05-27T12:01:00Z","level":"ERROR","msg":"Mod import failed","operation":"import_mod","event":"failed","source_path":"/Users/fergal/Games/Broken.zip"}
{"time":"2026-05-27T12:02:00Z","level":"INFO","msg":"Game scan completed","operation":"scan_games","event":"completed","inserted_count":1}
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manager, err := diagnostics.NewManager(diagnostics.Options{LogPath: logPath})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer closeServiceTestDiagnostics(t, manager)

	content, err := NewDiagnosticsService(manager).ListRecentRawLogs(context.Background(), dto.ListDiagnosticLogsInput{
		Limit:     1,
		Level:     "info",
		Operation: diagnostics.OperationImportMod,
	})
	if err != nil {
		t.Fatalf("ListRecentRawLogs() error = %v", err)
	}

	var entries []map[string]any
	if err := json.Unmarshal([]byte(content), &entries); err != nil {
		t.Fatalf("Unmarshal() error = %v, content = %q", err, content)
	}
	if len(entries) != 1 {
		t.Fatalf("ListRecentRawLogs() entries = %d, want 1 in %s", len(entries), content)
	}
	if entries[0]["source_path"] != "/Users/fergal/Games/Mod.zip" {
		t.Fatalf("ListRecentRawLogs() entry = %+v, want raw source path preserved", entries[0])
	}
	if !strings.Contains(content, "\n  {") {
		t.Fatalf("ListRecentRawLogs() content = %q, want pretty JSON", content)
	}
}

func TestDiagnosticsServiceExportLogsWritesVisibleEntries(t *testing.T) {
	t.Parallel()

	manager, err := diagnostics.NewManager(diagnostics.Options{
		LogPath: filepath.Join(t.TempDir(), diagnostics.DefaultLogFileName),
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer closeServiceTestDiagnostics(t, manager)

	service := NewDiagnosticsService(manager)
	exportPath := filepath.Join(t.TempDir(), "logs.txt")
	err = service.ExportLogs(context.Background(), dto.ExportDiagnosticLogsInput{
		Path: exportPath,
		Entries: []dto.DiagnosticLogEntry{
			{
				Timestamp: "2026-05-27T12:01:00Z",
				Level:     "error",
				Operation: diagnostics.OperationImportMod,
				Message:   "Mod import failed",
				Details: map[string]string{
					"Error": "permission denied",
					"Event": diagnostics.EventFailed,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ExportLogs() error = %v", err)
	}

	contents, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	got := string(contents)
	if !strings.Contains(got, "2026-05-27T12:01:00Z ERROR [import_mod] Mod import failed") ||
		!strings.Contains(got, "  Error: permission denied") ||
		!strings.Contains(got, "  Event: failed") {
		t.Fatalf("exported logs = %q, want formatted entry and details", got)
	}
}

func newServiceTestDiagnostics(t *testing.T) *diagnostics.Manager {
	t.Helper()

	manager, err := diagnostics.NewManager(diagnostics.Options{
		LogPath: filepath.Join(t.TempDir(), diagnostics.DefaultLogFileName),
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	return manager
}

func closeServiceTestDiagnostics(t *testing.T, manager *diagnostics.Manager) {
	t.Helper()

	if err := manager.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func readServiceTestLogs(t *testing.T, manager *diagnostics.Manager) []diagnostics.LogEntry {
	t.Helper()

	entries, err := manager.RecentLogs(context.Background(), diagnostics.RecentLogsInput{Limit: 50})
	if err != nil {
		t.Fatalf("RecentLogs() error = %v", err)
	}

	return entries
}

func findDiagnosticEvent(entries []diagnostics.LogEntry, operation string, event string) (diagnostics.LogEntry, bool) {
	for _, entry := range entries {
		if entry.Operation == operation && entry.Details["Event"] == event {
			return entry, true
		}
	}

	return diagnostics.LogEntry{}, false
}

func hasDiagnosticEvent(entries []diagnostics.LogEntry, operation string, event string) bool {
	_, ok := findDiagnosticEvent(entries, operation, event)
	return ok
}

func int64String(value int64) string {
	return fmt.Sprintf("%d", value)
}
