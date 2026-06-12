package storage

import (
	"context"
	"testing"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestOptiScalerTargetPersistenceUsesCaseInsensitiveRelativeIdentity(t *testing.T) {
	t.Parallel()
	store := openMigratedStore(t)
	defer closeStore(t, store)
	gameID := insertProfileTestGame(t, store, "Game", t.TempDir())

	input := dbtypes.SaveOptiScalerTargetInput{
		GameID: gameID, TargetRelativePath: `Binaries\Win64`,
		ExecutableRelativePath: `Binaries\Win64\Game.exe`, GraphicsAPI: "directx",
		ProxyFilename: "dxgi.dll", DXGISpoofing: true, ReleaseTag: "v1",
		ReleaseVersion: "v1a", ReleaseAssetName: "archive.7z", ReleaseDigest: "digest",
		ManagementOrigin: "installed", Status: "managed", ManifestJSON: `{"version":1}`,
		WarningVersion: "warning-v1",
	}
	if _, err := store.SaveOptiScalerTarget(context.Background(), input); err != nil {
		t.Fatalf("SaveOptiScalerTarget() error = %v", err)
	}
	input.TargetRelativePath = `binaries\win64`
	input.ProxyFilename = "winmm.dll"
	if _, err := store.SaveOptiScalerTarget(context.Background(), input); err != nil {
		t.Fatalf("SaveOptiScalerTarget() update error = %v", err)
	}
	targets, err := store.ListOptiScalerTargets(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListOptiScalerTargets() error = %v", err)
	}
	if len(targets) != 1 || targets[0].ProxyFilename != "winmm.dll" || targets[0].ProcessFilter != nil {
		t.Fatalf("ListOptiScalerTargets() = %+v", targets)
	}
}
