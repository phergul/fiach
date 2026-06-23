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

func TestOfficialInstallerManagedDXGIStaging(t *testing.T) {
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
	artifact := InstallerArtifact{
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
	}
	baseRequest := SetupRequest{
		Artifact:                        artifact,
		TargetExecutable:                targetExecutable,
		RenderingAPI:                    RenderingAPID3D11,
		Operation:                       SetupOperationInstall,
		Architecture:                    ArchitectureX64,
		ExpectedProxy:                   "dxgi.dll",
		AllowedProxyOutputRelativePaths: []string{"d3d11.dll"},
		ExpectedOutputRelativePaths: []string{
			"dxgi.dll",
			"ReShade.ini",
			"ReShade.log",
			"ReShadePreset.ini",
		},
	}

	clean, err := PrepareSetup(context.Background(), baseRequest, SetupRunnerOptions{
		WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("clean PrepareSetup() error = %v; execution = %+v", err, clean.Execution)
	}
	cleanProxy := preparedLiveProxyFile(t, clean)
	t.Logf("clean dxgi staging source: relative=%s actual=%s", cleanProxy.RelativePath, filepath.Base(cleanProxy.Path))

	requestWithExistingDXGI := baseRequest
	requestWithExistingDXGI.ExistingInputs = []SetupInput{
		{
			SourcePath:   cleanProxy.Path,
			RelativePath: "dxgi.dll",
		},
	}
	withExisting, err := PrepareSetup(context.Background(), requestWithExistingDXGI, SetupRunnerOptions{
		WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("existing dxgi PrepareSetup() error = %v; execution = %+v", err, withExisting.Execution)
	}
	existingProxy := preparedLiveProxyFile(t, withExisting)
	t.Logf("existing dxgi staging source: relative=%s actual=%s", existingProxy.RelativePath, filepath.Base(existingProxy.Path))
}

func preparedLiveProxyFile(t *testing.T, result SetupRunResult) PreparedSetupFile {
	t.Helper()
	if result.Prepared == nil || len(result.Prepared.Files) == 0 {
		t.Fatalf("Prepared = %+v", result.Prepared)
	}
	for _, file := range result.Prepared.Files {
		if file.RelativePath == "dxgi.dll" {
			return file
		}
	}
	t.Fatalf("Prepared files = %+v, want dxgi.dll", result.Prepared.Files)
	return PreparedSetupFile{}
}
