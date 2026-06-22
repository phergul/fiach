package services

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/optiscaler"
	"github.com/phergul/fiach/internal/reshade"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestReshadeServiceDetectGameReShadeReturnsUnsupportedWithoutStorageAccess(t *testing.T) {
	t.Parallel()

	service := NewReshadeService(nil, testLogger(), nil)
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

	service := NewReshadeService(store, testLogger(), nil)
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

	service := NewReshadeService(store, testLogger(), nil)
	service.operatingSystem = "windows"
	service.scan = func(string, []string) (reshade.Result, error) {
		return reshade.Result{Targets: []reshade.Target{{
			Path:        target,
			Executables: []string{filepath.Join(target, "Game.exe")},
		}}}, nil
	}

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

func TestReshadeServiceDiscoverManagedReShadeCandidatesReturnsPerFileWarnings(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Broken.exe"))
	gameID := insertServiceTestGame(t, store, "Portal", root)

	service := NewReshadeService(store, testLogger(), nil)
	service.operatingSystem = "windows"
	result, err := service.DiscoverManagedReShadeCandidates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("DiscoverManagedReShadeCandidates() error = %v", err)
	}
	if len(result.Candidates) != 0 || len(result.Warnings) != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestReshadeServiceListManagedReShadeChainTargetsCombinesProducts(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	root := t.TempDir()
	gameID := insertServiceTestGame(t, store, "Portal", root)
	if _, err := store.SaveOptiScalerTarget(context.Background(), dbtypes.SaveOptiScalerTargetInput{
		GameID:                 gameID,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		GraphicsAPI:            string(optiscaler.GraphicsAPIDirectX),
		ProxyFilename:          "dxgi.dll",
		ReleaseDigest:          "digest",
		ReleaseTag:             "v1",
		ReleaseVersion:         "1",
		ReleaseAssetName:       "OptiScaler.7z",
		ManagementOrigin:       "installed",
		Status:                 "managed",
		WarningVersion:         "1",
		ManifestJSON:           `{"version":1}`,
	}); err != nil {
		t.Fatalf("SaveOptiScalerTarget() error = %v", err)
	}
	manifest, err := json.Marshal(reshade.Manifest{Version: reshade.ManifestVersion})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveReShadeTarget(context.Background(), dbtypes.SaveReShadeTargetInput{
		GameID:                 gameID,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           string(reshade.RenderingAPID3D11),
		ProxyFilename:          "ReShade64.dll",
		Architecture:           string(reshade.ArchitectureX64),
		BuildVariant:           string(reshade.BuildVariantStandard),
		RuntimeVersion:         "6",
		ManagementOrigin:       "installed",
		Status:                 "managed",
		ManifestJSON:           string(manifest),
	}); err != nil {
		t.Fatalf("SaveReShadeTarget() error = %v", err)
	}

	service := NewReshadeService(store, testLogger(), nil)
	result, err := service.ListManagedReShadeChainTargets(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListManagedReShadeChainTargets() error = %v", err)
	}
	if len(result) != 1 || result[0].PrimaryOwner != "optiscaler" ||
		result[0].OptiScaler == nil || result[0].ReShade == nil {
		t.Fatalf("result = %+v", result)
	}
}
