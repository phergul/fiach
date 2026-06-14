//go:build windows

package reshade

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/fileops"
)

func TestOfficialInstallerManagedStaging(t *testing.T) {
	installerPath := os.Getenv("FIACH_RESHADE_INSTALLER_FIXTURE")
	if installerPath == "" {
		t.Skip("FIACH_RESHADE_INSTALLER_FIXTURE is not set")
	}
	installerPath, err := filepath.Abs(installerPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(installerPath); err != nil {
		t.Fatalf("installer fixture %q: %v", installerPath, err)
	}
	targetExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	signature, err := (platformInstallerSignatureVerifier{}).VerifyInstallerSignature(
		installerPath,
		InstallerVariantStandard,
	)
	if err != nil {
		t.Fatalf("VerifyInstallerSignature() error = %v", err)
	}
	hash, size, err := fileops.FileIntegrity(installerPath)
	if err != nil {
		t.Fatal(err)
	}

	result, err := PrepareSetup(context.Background(), SetupRequest{
		Artifact: InstallerArtifact{
			InstallerRelease: InstallerRelease{
				Version:   "6.7.3",
				Variant:   InstallerVariantStandard,
				AssetName: "ReShade_Setup_6.7.3.exe",
				URL:       "https://reshade.me/downloads/ReShade_Setup_6.7.3.exe",
			},
			Path:      installerPath,
			SizeBytes: size,
			SHA256:    hash,
			Signature: signature,
		},
		TargetExecutable: targetExecutable,
		RenderingAPI:     RenderingAPID3D11,
		Operation:        SetupOperationInstall,
		Architecture:     ArchitectureX64,
		ExpectedProxy:    "d3d11.dll",
		ExpectedOutputRelativePaths: []string{
			"d3d11.dll",
			"ReShade.ini",
			"ReShade.log",
			"ReShadePreset.ini",
		},
	}, SetupRunnerOptions{
		WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("PrepareSetup() error = %v; execution = %+v", err, result.Execution)
	}
	if result.Prepared == nil || len(result.Prepared.Files) == 0 {
		t.Fatalf("Prepared = %+v", result.Prepared)
	}
}
