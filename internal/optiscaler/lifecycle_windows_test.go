//go:build windows

package optiscaler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestManagerAdoptsVerifiedInstallationWithoutRollbackData(t *testing.T) {
	t.Parallel()

	gameRoot, request := newLifecycleGame(t)
	pkg := testPreparedPackageWithRuntime(t, "runtime-v1")
	writePackageToTarget(t, pkg, gameRoot, request.ProxyFilename)

	store := newMemoryStore()
	manager := lifecycleManager(t, store, pkg, func(string) (Ownership, error) {
		return OwnershipOptiScaler, nil
	})
	request.Action = ActionAdopt
	request.AcknowledgeWarning = true

	preview := previewAndApply(t, manager, gameRoot, request)
	if len(preview.Warnings) == 0 {
		t.Fatal("adoption preview has no missing rollback warning")
	}
	target, found, err := store.GetOptiScalerTarget(context.Background(), request.GameID, ".")
	if err != nil || !found {
		t.Fatalf("GetOptiScalerTarget() = %+v, %v, %v", target, found, err)
	}
	if target.ManagementOrigin != "adopted" {
		t.Fatalf("ManagementOrigin = %q, want adopted", target.ManagementOrigin)
	}
	manifest := mustManifest(t, target.ManifestJSON)
	if manifest.HasPreAdoptionRollbackData {
		t.Fatal("HasPreAdoptionRollbackData = true, want false")
	}
}

func TestManagerUpdateAndRepairPreserveINIAndRestoreFiles(t *testing.T) {
	t.Parallel()

	gameRoot, request := newLifecycleGame(t)
	store := newMemoryStore()
	pkgV1 := testPreparedPackageWithRuntime(t, "runtime-v1")
	managerV1 := lifecycleManager(t, store, pkgV1, nil)
	request.Action = ActionInstall
	request.AcknowledgeWarning = true
	previewAndApply(t, managerV1, gameRoot, request)

	iniPath := filepath.Join(gameRoot, "OptiScaler.ini")
	customINI := "; keep me\r\n[Custom]\r\nValue=42\r\n"
	if err := os.WriteFile(iniPath, []byte(customINI), 0o644); err != nil {
		t.Fatalf("WriteFile(OptiScaler.ini) error = %v", err)
	}
	pkgV2 := testPreparedPackageWithRuntime(t, "runtime-v2")
	managerV2 := lifecycleManager(t, store, pkgV2, func(path string) (Ownership, error) {
		if strings.EqualFold(filepath.Base(path), request.ProxyFilename) {
			return OwnershipOptiScaler, nil
		}
		return OwnershipUnknown, errors.New("unknown test file")
	})
	request.Action = ActionUpdate
	request.AcknowledgeWarning = false
	request.BackupAndContinue = true
	previewAndApply(t, managerV2, gameRoot, request)

	contents, err := os.ReadFile(iniPath)
	if err != nil {
		t.Fatalf("ReadFile(OptiScaler.ini) error = %v", err)
	}
	if !strings.Contains(string(contents), "; keep me") || !strings.Contains(string(contents), "Value=42") {
		t.Fatalf("updated INI did not preserve unrelated content:\n%s", contents)
	}
	if got, err := os.ReadFile(filepath.Join(gameRoot, request.ProxyFilename)); err != nil || string(got) != "runtime-v2" {
		t.Fatalf("updated proxy = %q, %v", got, err)
	}

	if err := os.Remove(filepath.Join(gameRoot, "LICENSE.txt")); err != nil {
		t.Fatalf("Remove(LICENSE.txt) error = %v", err)
	}
	request.Action = ActionRepair
	request.BackupAndContinue = true
	previewAndApply(t, managerV2, gameRoot, request)
	if got, err := os.ReadFile(filepath.Join(gameRoot, "LICENSE.txt")); err != nil || string(got) != "license" {
		t.Fatalf("repaired license = %q, %v", got, err)
	}
}

func TestManagerDirectXChainingAndUninstallRestoreReShadeProxy(t *testing.T) {
	t.Parallel()

	gameRoot, request := newLifecycleGame(t)
	proxyPath := filepath.Join(gameRoot, request.ProxyFilename)
	if err := os.WriteFile(proxyPath, []byte("reshade"), 0o644); err != nil {
		t.Fatalf("WriteFile(ReShade proxy) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(gameRoot, "ReShade64.dll"), []byte("older-reshade"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing chained ReShade) error = %v", err)
	}
	pkg := testPreparedPackageWithRuntime(t, "optiscaler")
	store := newMemoryStore()
	manager := lifecycleManager(t, store, pkg, func(path string) (Ownership, error) {
		switch strings.ToLower(filepath.Base(path)) {
		case strings.ToLower(request.ProxyFilename), "reshade64.dll":
			return OwnershipReShade, nil
		default:
			return OwnershipUnknown, errors.New("unknown test file")
		}
	})
	request.Action = ActionInstall
	request.AcknowledgeWarning = true
	request.EnableReShadeCoexistence = true
	preview := previewAndApply(t, manager, gameRoot, request)
	if preview.Operations[0].BackupPath == "" {
		t.Fatal("install preview does not expose the existing proxy backup path")
	}
	if _, err := os.Stat(preview.Operations[0].BackupPath); err != nil {
		t.Fatalf("planned proxy backup was not created: %v", err)
	}

	if got, err := os.ReadFile(filepath.Join(gameRoot, "ReShade64.dll")); err != nil || string(got) != "reshade" {
		t.Fatalf("chained ReShade = %q, %v", got, err)
	}
	if got, err := os.ReadFile(proxyPath); err != nil || string(got) != "optiscaler" {
		t.Fatalf("primary proxy = %q, %v", got, err)
	}

	request.Action = ActionUninstall
	request.AcknowledgeWarning = false
	request.EnableReShadeCoexistence = false
	previewAndApply(t, manager, gameRoot, request)
	if got, err := os.ReadFile(proxyPath); err != nil || string(got) != "reshade" {
		t.Fatalf("restored ReShade proxy = %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(gameRoot, "ReShade64.dll")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReShade64.dll remains after uninstall: %v", err)
	}
}

func TestManagerInstallAfterManagedReShadeChainsPersistedRuntime(t *testing.T) {
	t.Parallel()

	gameRoot, request := newLifecycleGame(t)
	proxyPath := filepath.Join(gameRoot, request.ProxyFilename)
	if err := os.WriteFile(proxyPath, []byte("reshade"), 0o644); err != nil {
		t.Fatalf("WriteFile(ReShade proxy) error = %v", err)
	}
	hash, size, err := fileIntegrity(proxyPath)
	if err != nil {
		t.Fatalf("fileIntegrity(ReShade proxy) error = %v", err)
	}
	store := newMemoryStore()
	if _, err := store.SaveReShadeTarget(context.Background(), dbtypes.SaveReShadeTargetInput{
		GameID:                 request.GameID,
		TargetRelativePath:     request.TargetRelativePath,
		ExecutableRelativePath: request.ExecutableRelativePath,
		RenderingAPI:           "d3d11",
		ProxyFilename:          request.ProxyFilename,
		ActiveRuntimeFilename:  request.ProxyFilename,
		Architecture:           "x64",
		BuildVariant:           "standard",
		RuntimeVersion:         "6.7.3",
		ManagementOrigin:       "installed",
		Status:                 "managed",
		ManifestJSON: fmt.Sprintf(
			`{"version":1,"files":[{"relativePath":%q,"sha256":%q,"sizeBytes":%d,"ownership":"managed"}],"variantProvenance":"verified"}`,
			request.ProxyFilename,
			hash,
			size,
		),
	}); err != nil {
		t.Fatalf("SaveReShadeTarget() error = %v", err)
	}

	manager := lifecycleManager(t, store, testPreparedPackageWithRuntime(t, "optiscaler"), func(path string) (Ownership, error) {
		if strings.EqualFold(filepath.Base(path), request.ProxyFilename) {
			return OwnershipReShade, nil
		}
		return OwnershipUnknown, errors.New("unknown test file")
	})
	request.Action = ActionInstall
	request.AcknowledgeWarning = true

	previewAndApply(t, manager, gameRoot, request)
	reShadeTarget, found, err := store.GetReShadeTarget(context.Background(), request.GameID, request.TargetRelativePath)
	if err != nil || !found {
		t.Fatalf("GetReShadeTarget() = %+v, %v, %v", reShadeTarget, found, err)
	}
	if reShadeTarget.ProxyFilename != request.ProxyFilename || reShadeTarget.ActiveRuntimeFilename != "ReShade64.dll" {
		t.Fatalf("ReShade filenames = preferred %q active %q", reShadeTarget.ProxyFilename, reShadeTarget.ActiveRuntimeFilename)
	}
	if !strings.Contains(reShadeTarget.ManifestJSON, "ReShade64.dll") {
		t.Fatalf("ReShade manifest was not chained: %s", reShadeTarget.ManifestJSON)
	}
	optiTarget, found, err := store.GetOptiScalerTarget(context.Background(), request.GameID, request.TargetRelativePath)
	if err != nil || !found {
		t.Fatalf("GetOptiScalerTarget() = %+v, %v, %v", optiTarget, found, err)
	}
	optiManifest := mustManifest(t, optiTarget.ManifestJSON)
	var chainedFile ManagedFile
	for _, file := range optiManifest.Files {
		if strings.EqualFold(filepath.Base(file.RelativePath), "ReShade64.dll") {
			chainedFile = file
			break
		}
	}
	if chainedFile.Ownership != string(OwnershipReShade) {
		t.Fatalf("chained runtime ownership = %q, want %q", chainedFile.Ownership, OwnershipReShade)
	}
	if drift, err := detectDrift(gameRoot, optiManifest); err != nil || len(drift) != 0 {
		t.Fatalf("OptiScaler drift = %+v, %v", drift, err)
	}
}

func TestManagerRejectsUnknownProxyAndVulkanReShadeCoexistence(t *testing.T) {
	t.Parallel()

	gameRoot, request := newLifecycleGame(t)
	if err := os.WriteFile(filepath.Join(gameRoot, request.ProxyFilename), []byte("unknown"), 0o644); err != nil {
		t.Fatalf("WriteFile(proxy) error = %v", err)
	}
	manager := lifecycleManager(t, newMemoryStore(), testPreparedPackageWithRuntime(t, "runtime"), func(string) (Ownership, error) {
		return OwnershipUnknown, nil
	})
	request.Action = ActionInstall
	request.AcknowledgeWarning = true
	preview, err := manager.Preview(context.Background(), gameRoot, request)
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if preview.CanApply || len(preview.Conflicts) == 0 {
		t.Fatalf("unknown proxy preview = %+v", preview)
	}

	if err := os.Remove(filepath.Join(gameRoot, request.ProxyFilename)); err != nil {
		t.Fatalf("Remove(proxy) error = %v", err)
	}
	request.GraphicsAPI = GraphicsAPIVulkan
	request.ProxyFilename = "winmm.dll"
	request.EnableReShadeCoexistence = true
	preview, err = manager.Preview(context.Background(), gameRoot, request)
	if err != nil {
		t.Fatalf("Vulkan Preview() error = %v", err)
	}
	if preview.CanApply || len(preview.Conflicts) == 0 {
		t.Fatalf("Vulkan coexistence preview = %+v", preview)
	}

	request.EnableReShadeCoexistence = false
	previewAndApply(t, manager, gameRoot, request)
	if got, err := os.ReadFile(filepath.Join(gameRoot, "winmm.dll")); err != nil || string(got) != "runtime" {
		t.Fatalf("Vulkan proxy = %q, %v", got, err)
	}
}

func TestManagerRejectsUnknownExistingChainedRuntime(t *testing.T) {
	t.Parallel()

	gameRoot, request := newLifecycleGame(t)
	if err := os.WriteFile(filepath.Join(gameRoot, request.ProxyFilename), []byte("reshade"), 0o644); err != nil {
		t.Fatalf("WriteFile(proxy) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(gameRoot, "ReShade64.dll"), []byte("unknown"), 0o644); err != nil {
		t.Fatalf("WriteFile(ReShade64.dll) error = %v", err)
	}
	manager := lifecycleManager(t, newMemoryStore(), testPreparedPackageWithRuntime(t, "runtime"), func(path string) (Ownership, error) {
		if strings.EqualFold(filepath.Base(path), request.ProxyFilename) {
			return OwnershipReShade, nil
		}
		return OwnershipUnknown, nil
	})
	request.Action = ActionInstall
	request.AcknowledgeWarning = true
	request.EnableReShadeCoexistence = true

	preview, err := manager.Preview(context.Background(), gameRoot, request)
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if preview.CanApply || len(preview.Conflicts) == 0 {
		t.Fatalf("unknown chained runtime preview = %+v", preview)
	}
}

func TestManagerOperationFailureRollsBackAndRollbackFailureRequiresRecovery(t *testing.T) {
	t.Parallel()

	gameRoot, request := newLifecycleGame(t)
	store := newMemoryStore()
	pkgV1 := testPreparedPackageWithRuntime(t, "runtime-v1")
	manager := lifecycleManager(t, store, pkgV1, nil)
	request.Action = ActionInstall
	request.AcknowledgeWarning = true
	previewAndApply(t, manager, gameRoot, request)

	pkgV2 := testPreparedPackageWithRuntime(t, "runtime-v2")
	calls := 0
	failing := lifecycleManagerWithOptions(t, store, pkgV2, ManagerOptions{
		InspectOwnership: func(string) (Ownership, error) { return OwnershipOptiScaler, nil },
		ExecuteOperation: func(operation Operation) error {
			calls++
			if calls == 2 {
				return errors.New("injected operation failure")
			}
			return executeOperation(operation)
		},
	})
	request.Action = ActionUpdate
	request.AcknowledgeWarning = false
	preview, err := failing.Preview(context.Background(), gameRoot, request)
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if _, err := failing.Apply(context.Background(), gameRoot, request, preview.PreviewHash); err == nil {
		t.Fatal("Apply() error = nil, want injected failure")
	}
	if got, err := os.ReadFile(filepath.Join(gameRoot, request.ProxyFilename)); err != nil || string(got) != "runtime-v1" {
		t.Fatalf("proxy after rollback = %q, %v", got, err)
	}

	recoveryManager := lifecycleManagerWithOptions(t, store, pkgV2, ManagerOptions{
		InspectOwnership:  func(string) (Ownership, error) { return OwnershipOptiScaler, nil },
		ExecuteOperation:  func(Operation) error { return errors.New("injected operation failure") },
		RollbackSnapshots: func([]journalSnapshot) error { return errors.New("injected rollback failure") },
	})
	preview, err = recoveryManager.Preview(context.Background(), gameRoot, request)
	if err != nil {
		t.Fatalf("recovery Preview() error = %v", err)
	}
	if _, err := recoveryManager.Apply(context.Background(), gameRoot, request, preview.PreviewHash); err == nil {
		t.Fatal("recovery Apply() error = nil, want injected failure")
	}
	state, err := recoveryManager.RecoveryState()
	if err != nil || !state.Required {
		t.Fatalf("RecoveryState() = %+v, %v", state, err)
	}
	target, found, err := store.GetOptiScalerTarget(context.Background(), request.GameID, ".")
	if err != nil || !found || target.Status != "recovery_required" {
		t.Fatalf("target after rollback failure = %+v, %v, %v", target, found, err)
	}
}

func newLifecycleGame(t *testing.T) (string, Request) {
	t.Helper()
	root := t.TempDir()
	copyCurrentExecutable(t, filepath.Join(root, "Game.exe"))
	process := "Game.exe"
	return root, Request{
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		GraphicsAPI:            GraphicsAPIDirectX,
		ProxyFilename:          "dxgi.dll",
		ProcessFilter:          &process,
	}
}

func lifecycleManager(t *testing.T, store *memoryStore, pkg Package, inspect func(string) (Ownership, error)) *Manager {
	t.Helper()
	return lifecycleManagerWithOptions(t, store, pkg, ManagerOptions{InspectOwnership: inspect})
}

func lifecycleManagerWithOptions(t *testing.T, store *memoryStore, pkg Package, options ManagerOptions) *Manager {
	t.Helper()
	options.DataDir = t.TempDir()
	options.CacheDir = t.TempDir()
	options.PreparePackage = func(context.Context) (Release, Package, error) {
		return Release{
			Tag:       "v1",
			Version:   "OptiScaler v1",
			AssetName: "OptiScaler.7z",
			Digest:    strings.Repeat("a", 64),
			Size:      1,
		}, pkg, nil
	}
	return NewManager(store, options)
}

func previewAndApply(t *testing.T, manager *Manager, gameRoot string, request Request) Preview {
	t.Helper()
	preview, err := manager.Preview(context.Background(), gameRoot, request)
	if err != nil {
		t.Fatalf("Preview(%s) error = %v", request.Action, err)
	}
	if !preview.CanApply {
		t.Fatalf("Preview(%s) conflicts = %v", request.Action, preview.Conflicts)
	}
	if _, err := manager.Apply(context.Background(), gameRoot, request, preview.PreviewHash); err != nil {
		t.Fatalf("Apply(%s) error = %v", request.Action, err)
	}
	return preview
}

func testPreparedPackageWithRuntime(t *testing.T, runtime string) Package {
	t.Helper()
	pkg := testPreparedPackage(t)
	for index := range pkg.Files {
		if !strings.EqualFold(filepath.Base(pkg.Files[index].RelativePath), "OptiScaler.dll") {
			continue
		}
		if err := os.WriteFile(pkg.Files[index].SourcePath, []byte(runtime), 0o644); err != nil {
			t.Fatalf("WriteFile(runtime) error = %v", err)
		}
		hash, size, err := fileIntegrity(pkg.Files[index].SourcePath)
		if err != nil {
			t.Fatalf("fileIntegrity(runtime) error = %v", err)
		}
		pkg.Files[index].SHA256 = hash
		pkg.Files[index].SizeBytes = size
	}
	return pkg
}

func writePackageToTarget(t *testing.T, pkg Package, target string, proxy string) {
	t.Helper()
	for _, file := range pkg.Files {
		name := filepath.Base(file.RelativePath)
		if strings.EqualFold(name, "OptiScaler.dll") {
			name = proxy
		}
		contents, err := os.ReadFile(file.SourcePath)
		if err != nil {
			t.Fatalf("ReadFile(package %q) error = %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(target, name), contents, 0o644); err != nil {
			t.Fatalf("WriteFile(target %q) error = %v", name, err)
		}
	}
}

func mustManifest(t *testing.T, value string) Manifest {
	t.Helper()
	var manifest Manifest
	if err := json.Unmarshal([]byte(value), &manifest); err != nil {
		t.Fatalf("Unmarshal(manifest) error = %v", err)
	}
	return manifest
}

func fileIntegrity(path string) (string, int64, error) {
	return fileops.FileIntegrity(path)
}
