package services

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/services/dto"
)

func TestReshadeServiceDetectGameReShadeReturnsUnsupportedWithoutStorageAccess(t *testing.T) {
	t.Parallel()

	service := NewReshadeService(nil)
	service.operatingSystem = "darwin"

	result, err := service.DetectGameReShade(context.Background(), 1)
	if err != nil {
		t.Fatalf("DetectGameReShade() error = %v", err)
	}
	if result.Status != dto.ReShadeDetectionStatusUnsupported {
		t.Fatalf("Status = %q, want %q", result.Status, dto.ReShadeDetectionStatusUnsupported)
	}
	if result.UnsupportedReason == nil || *result.UnsupportedReason == "" {
		t.Fatalf("UnsupportedReason = %v, want populated reason", result.UnsupportedReason)
	}
}

func TestReshadeServiceDetectGameReShadeValidatesInstallPath(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	filePath := filepath.Join(t.TempDir(), "Game.exe")
	writeFile(t, filePath)
	gameID := insertServiceTestGame(t, store, "Portal", filePath)

	service := NewReshadeService(store)
	service.operatingSystem = "windows"

	_, err := service.DetectGameReShade(context.Background(), gameID)
	if err == nil {
		t.Fatal("DetectGameReShade() error = nil, want error")
	}
	if !contains(err.Error(), "detect game ReShade runtime") || !contains(err.Error(), "not a directory") {
		t.Fatalf("DetectGameReShade() error = %q, want service and path context", err.Error())
	}
}

func TestReshadeServiceDetectGameReShadeReturnsDetectedTargets(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	root := t.TempDir()
	target := filepath.Join(root, "bin")
	mkdirAll(t, target)
	writeFile(t, filepath.Join(target, "Game.exe"))
	writeFile(t, filepath.Join(target, "dxgi.dll"))
	writeFile(t, filepath.Join(target, "ReShade.ini"))
	gameID := insertServiceTestGame(t, store, "Portal", root)

	service := NewReshadeService(store)
	service.operatingSystem = "windows"

	result, err := service.DetectGameReShade(context.Background(), gameID)
	if err != nil {
		t.Fatalf("DetectGameReShade() error = %v", err)
	}
	if result.Status != dto.ReShadeDetectionStatusInstalled {
		t.Fatalf("Status = %q, want %q", result.Status, dto.ReShadeDetectionStatusInstalled)
	}
	if len(result.Targets) != 1 {
		t.Fatalf("Targets length = %d, want 1", len(result.Targets))
	}
	if result.Targets[0].Path != target {
		t.Fatalf("Target path = %q, want %q", result.Targets[0].Path, target)
	}
	if len(result.Targets[0].Executables) != 1 || result.Targets[0].Executables[0] != filepath.Join(target, "Game.exe") {
		t.Fatalf("Executables = %#v, want Game.exe path", result.Targets[0].Executables)
	}
}

func TestReshadeServiceDownloadAndOpenInstallerReturnsUnsupportedWithoutLaunching(t *testing.T) {
	t.Parallel()

	service := NewReshadeService(nil)
	service.operatingSystem = "darwin"

	_, err := service.DownloadAndOpenReShadeInstaller(context.Background())
	if err == nil {
		t.Fatal("DownloadAndOpenReShadeInstaller() error = nil, want error")
	}
	if !contains(err.Error(), "download and open ReShade installer") || !contains(err.Error(), "only supported on Windows") {
		t.Fatalf("DownloadAndOpenReShadeInstaller() error = %q, want service and unsupported context", err.Error())
	}
}
