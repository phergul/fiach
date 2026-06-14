//go:build windows

package optiscaler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestReShadeSessionPersistsAndRepairsReclaimedProxy(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	dataDir := t.TempDir()
	writeSessionTestFile(t, filepath.Join(gameRoot, "Game.exe"), "exe")
	writeSessionTestFile(t, filepath.Join(gameRoot, "dxgi.dll"), "optiscaler")
	writeSessionTestFile(t, filepath.Join(gameRoot, "ReShade64.dll"), "old-reshade")
	writeSessionTestFile(t, filepath.Join(gameRoot, "OptiScaler.ini"), "[Plugins]\nLoadReshade=false\n")
	store := newMemoryStore()
	saveSessionTestTarget(t, store, GraphicsAPIDirectX)
	manager := newSessionTestManager(store, dataDir)

	state, err := manager.StartReShadeSession(context.Background(), gameRoot, ReShadeSessionRequest{
		GameID: 1, TargetRelativePath: ".", InstallerVariant: ReShadeInstallerVariantStandard,
	})
	if err != nil {
		t.Fatalf("StartReShadeSession() error = %v", err)
	}
	resumed, err := newSessionTestManager(store, dataDir).GetReShadeSession()
	if err != nil || resumed == nil || resumed.ID != state.ID {
		t.Fatalf("GetReShadeSession() = %+v, %v", resumed, err)
	}

	writeSessionTestFile(t, filepath.Join(gameRoot, "dxgi.dll"), "reshade")
	result, err := manager.RescanReShadeSession(context.Background(), gameRoot)
	if err != nil {
		t.Fatalf("RescanReShadeSession() error = %v", err)
	}
	if result.Outcome != ReShadeSessionOutcomeRepairRequired || result.Session == nil ||
		result.Session.Preview == nil {
		t.Fatalf("RescanReShadeSession() = %+v", result)
	}
	var chainedBackup string
	for _, operation := range result.Session.Preview.Operations {
		if operation.Type == "move" && filepath.Base(operation.TargetPath) == "ReShade64.dll" {
			chainedBackup = operation.BackupPath
		}
	}
	if chainedBackup == "" {
		t.Fatal("repair preview did not preserve the existing chained runtime")
	}
	if _, err := manager.ApplyReShadeRepair(context.Background(), gameRoot, "stale"); err == nil {
		t.Fatal("ApplyReShadeRepair(stale) error = nil")
	}
	previewHash := result.Session.Preview.PreviewHash
	applyResult, err := manager.ApplyReShadeRepair(context.Background(), gameRoot, previewHash)
	if err != nil {
		t.Fatalf("ApplyReShadeRepair() error = %v", err)
	}
	if !applyResult.Success {
		t.Fatalf("ApplyReShadeRepair() = %+v", applyResult)
	}
	assertSessionTestFile(t, filepath.Join(gameRoot, "dxgi.dll"), "optiscaler")
	assertSessionTestFile(t, filepath.Join(gameRoot, "ReShade64.dll"), "reshade")
	ini, err := os.ReadFile(filepath.Join(gameRoot, "OptiScaler.ini"))
	if err != nil || !strings.Contains(strings.ToLower(string(ini)), "loadreshade=true") {
		t.Fatalf("OptiScaler.ini = %q, %v", ini, err)
	}
	if session, err := manager.GetReShadeSession(); err != nil || session != nil {
		t.Fatalf("GetReShadeSession() after apply = %+v, %v", session, err)
	}
}

func TestReShadeSessionRejectsVulkanAndArchivesConflictOnCancel(t *testing.T) {
	t.Parallel()

	t.Run("vulkan", func(t *testing.T) {
		gameRoot := t.TempDir()
		writeSessionTestFile(t, filepath.Join(gameRoot, "Game.exe"), "exe")
		writeSessionTestFile(t, filepath.Join(gameRoot, "winmm.dll"), "optiscaler")
		store := newMemoryStore()
		saveSessionTestTarget(t, store, GraphicsAPIVulkan)
		manager := newSessionTestManager(store, t.TempDir())

		_, err := manager.StartReShadeSession(context.Background(), gameRoot, ReShadeSessionRequest{
			GameID: 1, TargetRelativePath: ".", InstallerVariant: ReShadeInstallerVariantStandard,
		})
		if err == nil || !strings.Contains(err.Error(), "Vulkan") {
			t.Fatalf("StartReShadeSession() error = %v", err)
		}
	})

	t.Run("conflict cancellation", func(t *testing.T) {
		gameRoot := t.TempDir()
		dataDir := t.TempDir()
		writeSessionTestFile(t, filepath.Join(gameRoot, "Game.exe"), "exe")
		writeSessionTestFile(t, filepath.Join(gameRoot, "dxgi.dll"), "optiscaler")
		store := newMemoryStore()
		saveSessionTestTarget(t, store, GraphicsAPIDirectX)
		manager := newSessionTestManager(store, dataDir)
		_, err := manager.StartReShadeSession(context.Background(), gameRoot, ReShadeSessionRequest{
			GameID: 1, TargetRelativePath: ".", InstallerVariant: ReShadeInstallerVariantAddon,
		})
		if err != nil {
			t.Fatalf("StartReShadeSession() error = %v", err)
		}
		writeSessionTestFile(t, filepath.Join(gameRoot, "dxgi.dll"), "unknown")

		result, err := manager.CancelReShadeSession(context.Background(), gameRoot)
		if err != nil {
			t.Fatalf("CancelReShadeSession() error = %v", err)
		}
		if result.Outcome != ReShadeSessionOutcomeCancelled {
			t.Fatalf("CancelReShadeSession() = %+v", result)
		}
		archives, err := os.ReadDir(filepath.Join(dataDir, "archives", "reshade-sessions"))
		if err != nil || len(archives) != 1 {
			t.Fatalf("archives = %v, %v", archives, err)
		}
		assertSessionTestFile(t, filepath.Join(gameRoot, "dxgi.dll"), "unknown")
	})
}

func TestReShadeRepairRollsBackOnFailure(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	writeSessionTestFile(t, filepath.Join(gameRoot, "Game.exe"), "exe")
	writeSessionTestFile(t, filepath.Join(gameRoot, "dxgi.dll"), "optiscaler")
	store := newMemoryStore()
	saveSessionTestTarget(t, store, GraphicsAPIDirectX)
	manager := newSessionTestManager(store, t.TempDir())
	_, err := manager.StartReShadeSession(context.Background(), gameRoot, ReShadeSessionRequest{
		GameID: 1, TargetRelativePath: ".", InstallerVariant: ReShadeInstallerVariantStandard,
	})
	if err != nil {
		t.Fatalf("StartReShadeSession() error = %v", err)
	}
	writeSessionTestFile(t, filepath.Join(gameRoot, "dxgi.dll"), "reshade")
	result, err := manager.RescanReShadeSession(context.Background(), gameRoot)
	if err != nil {
		t.Fatalf("RescanReShadeSession() error = %v", err)
	}
	manager.executeOperation = func(operation Operation) error {
		if operation.Type == "copy" {
			return errors.New("injected copy failure")
		}
		return executeOperation(operation)
	}
	applyResult, err := manager.ApplyReShadeRepair(
		context.Background(), gameRoot, result.Session.Preview.PreviewHash)
	if err == nil || !applyResult.RolledBack {
		t.Fatalf("ApplyReShadeRepair() = %+v, %v", applyResult, err)
	}
	assertSessionTestFile(t, filepath.Join(gameRoot, "dxgi.dll"), "reshade")
	if _, err := os.Stat(filepath.Join(gameRoot, "ReShade64.dll")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReShade64.dll exists after rollback: %v", err)
	}
}

func newSessionTestManager(store Store, dataDir string) *Manager {
	return NewManager(store, ManagerOptions{
		DataDir: dataDir,
		InspectOwnership: func(path string) (Ownership, error) {
			contents, err := os.ReadFile(path)
			if err != nil {
				return OwnershipUnknown, err
			}
			switch string(contents) {
			case "optiscaler":
				return OwnershipOptiScaler, nil
			case "reshade", "old-reshade":
				return OwnershipReShade, nil
			default:
				return OwnershipUnknown, nil
			}
		},
	})
}

func saveSessionTestTarget(t *testing.T, store Store, graphicsAPI GraphicsAPI) {
	t.Helper()
	proxy := "dxgi.dll"
	if graphicsAPI == GraphicsAPIVulkan {
		proxy = "winmm.dll"
	}
	process := "Game.exe"
	_, err := store.SaveOptiScalerTarget(context.Background(), dbtypes.SaveOptiScalerTargetInput{
		GameID: 1, TargetRelativePath: ".", ExecutableRelativePath: "Game.exe",
		GraphicsAPI: string(graphicsAPI), ProxyFilename: proxy, ProcessFilter: &process,
		ReleaseTag: "v1", ReleaseVersion: "OptiScaler v1", ReleaseAssetName: "OptiScaler.7z",
		ReleaseDigest: strings.Repeat("a", 64), ManagementOrigin: "installed", Status: "managed",
		ManifestJSON:   `{"version":1,"files":[],"config":{"loadReShade":false,"dxgiSpoofing":false,"targetProcessName":"Game.exe","checkForUpdate":false},"hasPreAdoptionRollbackData":true}`,
		WarningVersion: WarningVersion,
	})
	if err != nil {
		t.Fatalf("SaveOptiScalerTarget() error = %v", err)
	}
}

func writeSessionTestFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func assertSessionTestFile(t *testing.T, path string, want string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != want {
		t.Fatalf("ReadFile(%q) = %q, %v; want %q", path, contents, err, want)
	}
}
