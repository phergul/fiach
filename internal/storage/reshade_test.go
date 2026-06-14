package storage

import (
	"context"
	"testing"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestReShadeTargetPersistencePreservesNullableProvenance(t *testing.T) {
	t.Parallel()
	store := openMigratedStore(t)
	defer closeStore(t, store)
	gameID := insertProfileTestGame(t, store, "Game", t.TempDir())

	input := dbtypes.SaveReShadeTargetInput{
		GameID: gameID, TargetRelativePath: `Binaries\Win64`,
		ExecutableRelativePath: `Binaries\Win64\Game.exe`,
		RenderingAPI:           "d3d11", ProxyFilename: "dxgi.dll",
		Architecture: "x64", BuildVariant: "standard", RuntimeVersion: "6.5.1",
		ManagementOrigin: "adopted", Status: "managed",
		ManifestJSON: `{"version":1,"files":[],"hasPreAdoptionRollbackData":false}`,
	}
	if _, err := store.SaveReShadeTarget(context.Background(), input); err != nil {
		t.Fatalf("SaveReShadeTarget() error = %v", err)
	}
	input.TargetRelativePath = `binaries\win64`
	input.ProxyFilename = "d3d11.dll"
	if _, err := store.SaveReShadeTarget(context.Background(), input); err != nil {
		t.Fatalf("SaveReShadeTarget() update error = %v", err)
	}
	targets, err := store.ListReShadeTargets(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListReShadeTargets() error = %v", err)
	}
	if len(targets) != 1 || targets[0].ProxyFilename != "d3d11.dll" {
		t.Fatalf("ListReShadeTargets() = %+v", targets)
	}
	target := targets[0]
	if target.InstallerTag != nil || target.InstallerAssetName != nil ||
		target.InstallerURL != nil || target.InstallerDigest != nil || target.InstallerSize != nil {
		t.Fatalf("nullable provenance collapsed: %+v", target)
	}
}

func TestReShadeTargetPersistenceRejectsInvalidValues(t *testing.T) {
	t.Parallel()
	store := openMigratedStore(t)
	defer closeStore(t, store)
	gameID := insertProfileTestGame(t, store, "Game", t.TempDir())
	input := dbtypes.SaveReShadeTargetInput{
		GameID: gameID, TargetRelativePath: `..\outside`,
		ExecutableRelativePath: `Game.exe`, RenderingAPI: "directx",
		ProxyFilename: "dxgi.dll", Architecture: "x64", BuildVariant: "standard",
		RuntimeVersion: "6", ManagementOrigin: "installed", Status: "managed",
		ManifestJSON: `{"version":1}`,
	}
	if _, err := store.SaveReShadeTarget(context.Background(), input); err == nil {
		t.Fatal("SaveReShadeTarget() error = nil")
	}
}
