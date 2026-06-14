//go:build windows

package optiscaler

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestManagerInstallApplyPersistsManifestAndDriftBlocksUninstall(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	executablePath := filepath.Join(gameRoot, "Game.exe")
	copyCurrentExecutable(t, executablePath)
	pkg := testPreparedPackage(t)

	store := newMemoryStore()
	manager := NewManager(store, ManagerOptions{
		DataDir:  t.TempDir(),
		CacheDir: t.TempDir(),
		PreparePackage: func(context.Context) (Release, Package, error) {
			return Release{
				Tag:       "v1",
				Version:   "OptiScaler v1a",
				AssetName: "Optiscaler_v1-final.7z",
				Digest:    strings.Repeat("a", 64),
				Size:      1,
			}, pkg, nil
		},
	})
	process := "Game.exe"
	request := Request{
		Action:                 ActionInstall,
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		GraphicsAPI:            GraphicsAPIDirectX,
		ProxyFilename:          "dxgi.dll",
		ProcessFilter:          &process,
		AcknowledgeWarning:     true,
	}
	preview, err := manager.Preview(context.Background(), gameRoot, request)
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if !preview.CanApply || preview.PreviewHash == "" {
		t.Fatalf("Preview() = %+v, want applicable hashed preview", preview)
	}
	if _, err := manager.Apply(context.Background(), gameRoot, request, "stale"); err == nil {
		t.Fatal("Apply() stale hash error = nil")
	}
	result, err := manager.Apply(context.Background(), gameRoot, request, preview.PreviewHash)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !result.Success {
		t.Fatalf("Apply() = %+v", result)
	}
	for _, name := range []string{"dxgi.dll", "OptiScaler.ini", "LICENSE.txt"} {
		if _, err := os.Stat(filepath.Join(gameRoot, name)); err != nil {
			t.Fatalf("installed file %s: %v", name, err)
		}
	}
	target, found, err := store.GetOptiScalerTarget(context.Background(), 1, ".")
	if err != nil || !found || !strings.Contains(target.ManifestJSON, `"version":1`) {
		t.Fatalf("persisted target = %+v, %v, %v", target, found, err)
	}

	if err := os.WriteFile(filepath.Join(gameRoot, "dxgi.dll"), []byte("drift"), 0o644); err != nil {
		t.Fatalf("write drift: %v", err)
	}
	uninstall := request
	uninstall.Action = ActionUninstall
	uninstall.AcknowledgeWarning = false
	uninstallPreview, err := manager.Preview(context.Background(), gameRoot, uninstall)
	if err != nil {
		t.Fatalf("uninstall Preview() error = %v", err)
	}
	if uninstallPreview.CanApply || len(uninstallPreview.Drift) == 0 {
		t.Fatalf("uninstall Preview() = %+v, want blocking drift", uninstallPreview)
	}
	uninstall.BackupAndContinue = true
	uninstallPreview, err = manager.Preview(context.Background(), gameRoot, uninstall)
	if err != nil {
		t.Fatalf("backup-and-continue Preview() error = %v", err)
	}
	if !uninstallPreview.CanApply {
		t.Fatalf("backup-and-continue Preview() = %+v, want applicable preview", uninstallPreview)
	}
	if _, err := manager.Apply(context.Background(), gameRoot, uninstall, uninstallPreview.PreviewHash); err != nil {
		t.Fatalf("uninstall Apply() error = %v", err)
	}
	if _, found, err := store.GetOptiScalerTarget(context.Background(), 1, "."); err != nil || found {
		t.Fatalf("target after uninstall found = %v, error = %v", found, err)
	}
	for _, name := range []string{"dxgi.dll", "OptiScaler.ini", "LICENSE.txt"} {
		if _, err := os.Stat(filepath.Join(gameRoot, name)); !os.IsNotExist(err) {
			t.Fatalf("uninstalled file %s still exists, error = %v", name, err)
		}
	}
}

func testPreparedPackage(t *testing.T) Package {
	t.Helper()
	root := t.TempDir()
	var files []PackageFile
	for name, contents := range map[string]string{
		"OptiScaler.dll": "runtime",
		"OptiScaler.ini": "[Custom]\nValue=kept\n",
		"LICENSE.txt":    "license",
	} {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write package file: %v", err)
		}
		hash, size, err := fileops.FileIntegrity(path)
		if err != nil {
			t.Fatalf("hash package file: %v", err)
		}
		files = append(files, PackageFile{
			RelativePath: name,
			SourcePath:   path,
			SHA256:       hash,
			SizeBytes:    size,
		})
	}
	return Package{
		Root:  root,
		Files: files,
	}
}

func copyCurrentExecutable(t *testing.T, destination string) {
	t.Helper()
	sourcePath, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatalf("open current executable: %v", err)
	}
	defer source.Close()
	target, err := os.Create(destination)
	if err != nil {
		t.Fatalf("create executable copy: %v", err)
	}
	if _, err := io.Copy(target, source); err != nil {
		_ = target.Close()
		t.Fatalf("copy executable: %v", err)
	}
	if err := target.Close(); err != nil {
		t.Fatalf("close executable copy: %v", err)
	}
}
