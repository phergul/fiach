package desired_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/desired"
	"github.com/phergul/fiach/internal/deployment/rules"
	"github.com/phergul/fiach/internal/installconfig"
	"github.com/phergul/fiach/internal/operationplan"
	"github.com/phergul/fiach/internal/storage"
)

func TestBuildDesiredState_LoadOrderWinner(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	store := openDesiredStore(t)
	defer closeDesiredStore(t, store)

	gameID := insertDesiredGame(t, store, "Skyrim", gameRoot)
	profileID := insertDesiredProfile(t, store, gameID, "Default")

	firstSource := makeDesiredSourceTree(t, map[string]string{"plugin.txt": "alpha"})
	secondSource := makeDesiredSourceTree(t, map[string]string{"plugin.txt": "beta"})

	firstModID := insertDesiredMod(t, store, gameID, "Alpha", firstSource)
	secondModID := insertDesiredMod(t, store, gameID, "Beta", secondSource)

	addDesiredProfileMod(t, store, profileID, firstModID, true, 0)
	addDesiredProfileMod(t, store, profileID, secondModID, true, 1)
	addDesiredInstallConfig(t, store, firstModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Shared", nil)
	addDesiredInstallConfig(t, store, secondModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Shared", nil)

	resolved := resolveDesiredProfilePlan(t, store, profileID)
	state := buildDesiredState(t, resolved)

	file := desiredFile(t, state, "shared/plugin.txt")
	if file.SHA256 != sha256String("beta") {
		t.Fatalf("winner hash = %q, want beta content hash", file.SHA256)
	}
	if file.Winner.ModName != "Beta" || !file.Winner.IsWinner {
		t.Fatalf("winner = %+v, want Beta as winner", file.Winner)
	}
	if file.ConflictCategory != deployment.ConflictExpectedOverwrite {
		t.Fatalf("conflict category = %q, want expected_overwrite", file.ConflictCategory)
	}
}

func TestBuildDesiredState_MultiWriterStack(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	store := openDesiredStore(t)
	defer closeDesiredStore(t, store)

	gameID := insertDesiredGame(t, store, "Skyrim", gameRoot)
	profileID := insertDesiredProfile(t, store, gameID, "Default")

	firstSource := makeDesiredSourceTree(t, map[string]string{"plugin.txt": "alpha"})
	secondSource := makeDesiredSourceTree(t, map[string]string{"plugin.txt": "beta"})

	firstModID := insertDesiredMod(t, store, gameID, "Alpha", firstSource)
	secondModID := insertDesiredMod(t, store, gameID, "Beta", secondSource)

	addDesiredProfileMod(t, store, profileID, firstModID, true, 0)
	addDesiredProfileMod(t, store, profileID, secondModID, true, 1)
	addDesiredInstallConfig(t, store, firstModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Shared", nil)
	addDesiredInstallConfig(t, store, secondModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Shared", nil)

	resolved := resolveDesiredProfilePlan(t, store, profileID)
	state := buildDesiredState(t, resolved)

	file := desiredFile(t, state, "shared/plugin.txt")
	modWriters := filterModWriters(file.Writers)
	if len(modWriters) != 2 {
		t.Fatalf("mod writer count = %d, want 2: %+v", len(modWriters), file.Writers)
	}

	winnerCount := 0
	wouldWriteCount := 0
	for _, writer := range modWriters {
		if writer.IsWinner {
			winnerCount++
		}
		if writer.WouldWrite {
			wouldWriteCount++
		}
	}
	if winnerCount != 1 || wouldWriteCount != 1 {
		t.Fatalf("writers = %+v, want one winner and one would-write loser", modWriters)
	}
}

func TestBuildDesiredState_AmbiguousTiedLoadOrder(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	store := openDesiredStore(t)
	defer closeDesiredStore(t, store)

	gameID := insertDesiredGame(t, store, "Skyrim", gameRoot)
	profileID := insertDesiredProfile(t, store, gameID, "Default")

	firstSource := makeDesiredSourceTree(t, map[string]string{"plugin.txt": "alpha"})
	secondSource := makeDesiredSourceTree(t, map[string]string{"plugin.txt": "beta"})

	firstModID := insertDesiredMod(t, store, gameID, "Alpha", firstSource)
	secondModID := insertDesiredMod(t, store, gameID, "Beta", secondSource)

	addDesiredProfileMod(t, store, profileID, firstModID, true, 0)
	addDesiredProfileMod(t, store, profileID, secondModID, true, 0)
	addDesiredInstallConfig(t, store, firstModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Shared", nil)
	addDesiredInstallConfig(t, store, secondModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Shared", nil)

	resolved := resolveDesiredProfilePlan(t, store, profileID)
	state := buildDesiredState(t, resolved)

	file := desiredFile(t, state, "shared/plugin.txt")
	if file.ConflictCategory != deployment.ConflictAmbiguousOverwrite {
		t.Fatalf("conflict category = %q, want ambiguous_overwrite", file.ConflictCategory)
	}
	if file.FileStatus != deployment.FileStatusBlocked {
		t.Fatalf("file status = %q, want blocked", file.FileStatus)
	}
	if state.CanPreview() {
		t.Fatal("CanPreview() = true, want false for ambiguous overwrite")
	}
}

func TestBuildDesiredState_PerFileWinnerRuleOverridesLoadOrder(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	store := openDesiredStore(t)
	defer closeDesiredStore(t, store)

	gameID := insertDesiredGame(t, store, "Skyrim", gameRoot)
	profileID := insertDesiredProfile(t, store, gameID, "Default")

	firstSource := makeDesiredSourceTree(t, map[string]string{"plugin.txt": "alpha"})
	secondSource := makeDesiredSourceTree(t, map[string]string{"plugin.txt": "beta"})

	firstModID := insertDesiredMod(t, store, gameID, "Alpha", firstSource)
	secondModID := insertDesiredMod(t, store, gameID, "Beta", secondSource)

	addDesiredProfileMod(t, store, profileID, firstModID, true, 0)
	addDesiredProfileMod(t, store, profileID, secondModID, true, 1)
	addDesiredInstallConfig(t, store, firstModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Shared", nil)
	addDesiredInstallConfig(t, store, secondModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Shared", nil)

	resolved := resolveDesiredProfilePlan(t, store, profileID)
	state, err := desired.BuildDesiredState(context.Background(), resolved, []rules.DeploymentRule{
		{
			ProfileID:        profileID,
			GameRelativePath: "Shared/plugin.txt",
			RuleKind:         rules.RuleKindPerFileWinner,
			WinnerModID:      firstModID,
		},
	})
	if err != nil {
		t.Fatalf("BuildDesiredState() error = %v", err)
	}

	file := desiredFile(t, state, "shared/plugin.txt")
	if file.SHA256 != sha256String("alpha") {
		t.Fatalf("winner hash = %q, want alpha content hash", file.SHA256)
	}
	if file.Winner.ModName != "Alpha" || !file.Winner.IsWinner {
		t.Fatalf("winner = %+v, want Alpha as rule winner", file.Winner)
	}
	if file.FileStatus == deployment.FileStatusBlocked {
		t.Fatalf("file status = %q, want unblocked after rule", file.FileStatus)
	}
	if !strings.Contains(file.Explanation, "per-file rule") {
		t.Fatalf("explanation = %q, want per-file rule wording", file.Explanation)
	}
}

func TestBuildDesiredState_DestructiveFileDirectory(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	store := openDesiredStore(t)
	defer closeDesiredStore(t, store)

	gameID := insertDesiredGame(t, store, "Skyrim", gameRoot)
	profileID := insertDesiredProfile(t, store, gameID, "Default")

	fileSource := makeDesiredSourceTree(t, map[string]string{"Shared": "file-at-shared"})
	nestedSource := makeDesiredSourceTree(t, map[string]string{"nested.txt": "nested"})

	fileModID := insertDesiredMod(t, store, gameID, "FileMod", fileSource)
	nestedModID := insertDesiredMod(t, store, gameID, "NestedMod", nestedSource)

	addDesiredProfileMod(t, store, profileID, fileModID, true, 0)
	addDesiredProfileMod(t, store, profileID, nestedModID, true, 1)
	addDesiredInstallConfig(t, store, fileModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, ".", nil)
	addDesiredInstallConfig(t, store, nestedModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Shared", nil)

	resolved := resolveDesiredProfilePlan(t, store, profileID)
	state := buildDesiredState(t, resolved)

	fileEntry := desiredFile(t, state, "shared")
	nestedEntry := desiredFile(t, state, "shared/nested.txt")
	if fileEntry.ConflictCategory != deployment.ConflictDestructiveFileDirectory {
		t.Fatalf("file entry category = %q, want destructive_file_directory", fileEntry.ConflictCategory)
	}
	if nestedEntry.ConflictCategory != deployment.ConflictDestructiveFileDirectory {
		t.Fatalf("nested entry category = %q, want destructive_file_directory", nestedEntry.ConflictCategory)
	}
	if state.CanPreview() {
		t.Fatal("CanPreview() = true, want false for destructive file-directory conflict")
	}
}

func TestBuildDesiredState_BaseGameWriter(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	store := openDesiredStore(t)
	defer closeDesiredStore(t, store)

	gameID := insertDesiredGame(t, store, "Skyrim", gameRoot)
	profileID := insertDesiredProfile(t, store, gameID, "Default")

	source := makeDesiredSourceTree(t, map[string]string{"new.esp": "modded"})
	modID := insertDesiredMod(t, store, gameID, "Mod", source)
	addDesiredProfileMod(t, store, profileID, modID, true, 0)
	addDesiredInstallConfig(t, store, modID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Data", nil)

	resolved := resolveDesiredProfilePlan(t, store, profileID)
	state := buildDesiredState(t, resolved)

	file := desiredFile(t, state, "data/new.esp")
	if file.FileStatus != deployment.FileStatusAdded {
		t.Fatalf("file status = %q, want added", file.FileStatus)
	}
	for _, writer := range file.Writers {
		if writer.SourceKind == deployment.SourceKindBaseGame {
			t.Fatalf("writers = %+v, want no base_game writer", file.Writers)
		}
	}
}

func TestBuildDesiredState_NoFilesystemMutations(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	existingPath := filepath.Join(gameRoot, "Data", "existing.esp")
	if err := os.MkdirAll(filepath.Dir(existingPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(existingPath, []byte("vanilla"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	store := openDesiredStore(t)
	defer closeDesiredStore(t, store)

	gameID := insertDesiredGame(t, store, "Skyrim", gameRoot)
	profileID := insertDesiredProfile(t, store, gameID, "Default")

	source := makeDesiredSourceTree(t, map[string]string{"existing.esp": "modded", "new.esp": "added"})
	beforeGame := dirSnapshot(t, gameRoot)
	beforeSource := dirSnapshot(t, source)

	modID := insertDesiredMod(t, store, gameID, "Mod", source)
	addDesiredProfileMod(t, store, profileID, modID, true, 0)
	addDesiredInstallConfig(t, store, modID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Data", nil)

	resolved := resolveDesiredProfilePlan(t, store, profileID)
	_ = buildDesiredState(t, resolved)

	afterGame := dirSnapshot(t, gameRoot)
	afterSource := dirSnapshot(t, source)
	if beforeGame != afterGame {
		t.Fatalf("game install changed after desired-state build")
	}
	if beforeSource != afterSource {
		t.Fatalf("managed source changed after desired-state build")
	}
}

func TestBuildDesiredState_DisabledModExcluded(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	store := openDesiredStore(t)
	defer closeDesiredStore(t, store)

	gameID := insertDesiredGame(t, store, "Skyrim", gameRoot)
	profileID := insertDesiredProfile(t, store, gameID, "Default")

	enabledSource := makeDesiredSourceTree(t, map[string]string{"enabled.esp": "enabled"})
	disabledSource := makeDesiredSourceTree(t, map[string]string{"disabled.esp": "disabled"})

	enabledModID := insertDesiredMod(t, store, gameID, "Enabled", enabledSource)
	disabledModID := insertDesiredMod(t, store, gameID, "Disabled", disabledSource)

	addDesiredProfileMod(t, store, profileID, enabledModID, true, 0)
	addDesiredProfileMod(t, store, profileID, disabledModID, false, 1)
	addDesiredInstallConfig(t, store, enabledModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Data", nil)
	addDesiredInstallConfig(t, store, disabledModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Data", nil)

	resolved := resolveDesiredProfilePlan(t, store, profileID)
	state := buildDesiredState(t, resolved)

	if _, found := state.Files["data/disabled.esp"]; found {
		t.Fatal("disabled mod path present in desired state")
	}
	desiredFile(t, state, "data/enabled.esp")
}

func TestBuildDesiredState_CaseFoldingMerge(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	store := openDesiredStore(t)
	defer closeDesiredStore(t, store)

	gameID := insertDesiredGame(t, store, "Skyrim", gameRoot)
	profileID := insertDesiredProfile(t, store, gameID, "Default")

	firstSource := makeDesiredSourceTree(t, map[string]string{"Foo.txt": "alpha"})
	secondSource := makeDesiredSourceTree(t, map[string]string{"foo.txt": "beta"})

	firstModID := insertDesiredMod(t, store, gameID, "Alpha", firstSource)
	secondModID := insertDesiredMod(t, store, gameID, "Beta", secondSource)

	addDesiredProfileMod(t, store, profileID, firstModID, true, 0)
	addDesiredProfileMod(t, store, profileID, secondModID, true, 1)
	addDesiredInstallConfig(t, store, firstModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "Data", nil)
	addDesiredInstallConfig(t, store, secondModID, string(installconfig.StrategyTypeGenericCopy), installconfig.TargetBaseGameRoot, "data", nil)

	resolved := resolveDesiredProfilePlan(t, store, profileID)
	state := buildDesiredState(t, resolved)

	if len(state.Files) != 1 {
		t.Fatalf("file count = %d, want 1 merged canonical path", len(state.Files))
	}

	file := desiredFile(t, state, "data/foo.txt")
	if file.Winner.ModName != "Beta" {
		t.Fatalf("winner = %+v, want Beta from merged case-folded path", file.Winner)
	}
}

func buildDesiredState(t *testing.T, resolved operationplan.ResolveProfilePlanResult) deployment.DesiredState {
	t.Helper()

	state, err := desired.BuildDesiredState(context.Background(), resolved, nil)
	if err != nil {
		t.Fatalf("BuildDesiredState() error = %v", err)
	}
	return state
}

func resolveDesiredProfilePlan(t *testing.T, store *storage.Store, profileID int64) operationplan.ResolveProfilePlanResult {
	t.Helper()

	resolved, err := operationplan.ResolveProfilePlan(context.Background(), store, profileID)
	if err != nil {
		t.Fatalf("ResolveProfilePlan() error = %v", err)
	}
	return resolved
}

func desiredFile(t *testing.T, state deployment.DesiredState, canonicalPath string) deployment.DesiredFile {
	t.Helper()

	file, found := state.Files[canonicalPath]
	if !found {
		t.Fatalf("desired file %q not found in state: %+v", canonicalPath, state.Files)
	}
	return file
}

func filterModWriters(writers []deployment.WriterEntry) []deployment.WriterEntry {
	result := make([]deployment.WriterEntry, 0, len(writers))
	for _, writer := range writers {
		if writer.SourceKind == deployment.SourceKindMod {
			result = append(result, writer)
		}
	}
	return result
}

func sha256String(contents string) string {
	sum := sha256.Sum256([]byte(contents))
	return hex.EncodeToString(sum[:])
}

func dirSnapshot(t *testing.T, root string) string {
	t.Helper()

	var builder strings.Builder
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if relative == "." {
			return nil
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}

		builder.WriteString(relative)
		builder.WriteByte('|')
		builder.WriteString(info.Mode().String())
		builder.WriteByte('|')
		builder.WriteString(strconv.FormatInt(info.Size(), 10))
		builder.WriteByte('\n')
		return nil
	})
	if err != nil {
		t.Fatalf("filepath.WalkDir(%q) error = %v", root, err)
	}

	return builder.String()
}

func openDesiredStore(t *testing.T) *storage.Store {
	t.Helper()

	store, err := storage.Open(context.Background(), storage.Options{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("storage.Open() error = %v", err)
	}
	if err := store.MigrateUp(); err != nil {
		t.Fatalf("store.MigrateUp() error = %v", err)
	}

	return store
}

func closeDesiredStore(t *testing.T, store *storage.Store) {
	t.Helper()

	if err := store.Close(); err != nil {
		t.Fatalf("store.Close() error = %v", err)
	}
}

func insertDesiredGame(t *testing.T, store *storage.Store, name string, installPath string) int64 {
	t.Helper()

	result, err := store.DB().Exec(`
		INSERT INTO games (name, install_path)
		VALUES (?, ?)
	`, name, installPath)
	if err != nil {
		t.Fatalf("insert desired game: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("desired game LastInsertId(): %v", err)
	}

	return id
}

func insertDesiredProfile(t *testing.T, store *storage.Store, gameID int64, name string) int64 {
	t.Helper()

	result, err := store.DB().Exec(`
		INSERT INTO profiles (game_id, name)
		VALUES (?, ?)
	`, gameID, name)
	if err != nil {
		t.Fatalf("insert desired profile: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("desired profile LastInsertId(): %v", err)
	}

	return id
}

func insertDesiredMod(t *testing.T, store *storage.Store, gameID int64, name string, sourcePath string) int64 {
	t.Helper()

	originalSourcePath := filepath.Join("/imports", strings.ToLower(strings.ReplaceAll(name, " ", "-")))
	result, err := store.DB().Exec(`
		INSERT INTO mods (game_id, name, source_path, original_source_path)
		VALUES (?, ?, ?, ?)
	`, gameID, name, sourcePath, originalSourcePath)
	if err != nil {
		t.Fatalf("insert desired mod: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("desired mod LastInsertId(): %v", err)
	}

	return id
}

func addDesiredProfileMod(t *testing.T, store *storage.Store, profileID int64, modID int64, enabled bool, loadOrder int64) {
	t.Helper()

	enabledValue := 0
	if enabled {
		enabledValue = 1
	}

	if _, err := store.DB().Exec(`
		INSERT INTO profile_mods (profile_id, mod_id, enabled, load_order)
		VALUES (?, ?, ?, ?)
	`, profileID, modID, enabledValue, loadOrder); err != nil {
		t.Fatalf("insert desired profile mod: %v", err)
	}
}

func addDesiredInstallConfig(t *testing.T, store *storage.Store, modID int64, strategyType string, targetBase string, targetRelativePath string, sourceSubpath *string) {
	t.Helper()

	if _, err := store.DB().Exec(`
		INSERT INTO mod_install_configs (mod_id, strategy_type, target_base, target_relative_path, source_subpath)
		VALUES (?, ?, ?, ?, ?)
	`, modID, strategyType, targetBase, targetRelativePath, sourceSubpath); err != nil {
		t.Fatalf("insert desired install config: %v", err)
	}
}

func makeDesiredSourceTree(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for relativePath, contents := range files {
		path := filepath.Join(root, relativePath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", path, err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", path, err)
		}
	}

	return root
}
