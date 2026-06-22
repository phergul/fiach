package reshade

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type signatureVerifierFunc func(string, InstallerVariant) (InstallerSignature, error)

func (function signatureVerifierFunc) VerifyInstallerSignature(
	path string,
	variant InstallerVariant,
) (InstallerSignature, error) {
	return function(path, variant)
}

func TestResolveLatestInstallerReturnsPinnedRelease(t *testing.T) {
	t.Parallel()

	release, err := ResolveLatestInstaller(
		context.Background(),
		InstallerVariantAddon,
		InstallerResolveOptions{
			ManifestJSON: reShadeManifest(
				"6.10.0",
				"ReShade_Setup_6.10.0.exe",
				"https://reshade.me/downloads/ReShade_Setup_6.10.0.exe",
				strings.Repeat("a", 64),
				100,
				"6.10.0",
				"ReShade_Setup_6.10.0_Addon.exe",
				"https://reshade.me/downloads/ReShade_Setup_6.10.0_Addon.exe",
				strings.Repeat("b", 64),
				200,
			),
		},
	)
	if err != nil {
		t.Fatalf("ResolveLatestInstaller() error = %v", err)
	}
	if release.Version != "6.10.0" ||
		release.AssetName != "ReShade_Setup_6.10.0_Addon.exe" ||
		release.URL != "https://reshade.me/downloads/ReShade_Setup_6.10.0_Addon.exe" ||
		release.SHA256 != strings.Repeat("b", 64) ||
		release.SizeBytes != 200 {
		t.Fatalf("release = %+v", release)
	}
}

func TestResolveLatestInstallerRejectsUntrustedDownloadHost(t *testing.T) {
	t.Parallel()

	_, err := ResolveLatestInstaller(
		context.Background(),
		InstallerVariantStandard,
		InstallerResolveOptions{
			ManifestJSON: reShadeManifest(
				"6.7.3",
				"ReShade_Setup_6.7.3.exe",
				"https://example.invalid/downloads/ReShade_Setup_6.7.3.exe",
				strings.Repeat("a", 64),
				100,
				"6.7.3",
				"ReShade_Setup_6.7.3_Addon.exe",
				"https://reshade.me/downloads/ReShade_Setup_6.7.3_Addon.exe",
				strings.Repeat("b", 64),
				200,
			),
		},
	)
	if err == nil || !strings.Contains(err.Error(), "is not trusted") {
		t.Fatalf("ResolveLatestInstaller() error = %v, want untrusted host", err)
	}
}

func reShadeManifest(
	standardVersion string,
	standardAssetName string,
	standardURL string,
	standardDigest string,
	standardSize int64,
	addonVersion string,
	addonAssetName string,
	addonURL string,
	addonDigest string,
	addonSize int64,
) []byte {
	return []byte(fmt.Sprintf(`{
  "version": 1,
  "updatedAt": "2026-06-22T00:00:00Z",
  "tools": {
    "optiscaler": {
      "tag": "v0.9.3",
      "version": "OptiScaler v0.9.3",
      "assetName": "Optiscaler_0.9.3-final.20260618.7z",
      "url": "https://github.com/optiscaler/OptiScaler/releases/download/v0.9.3/Optiscaler_0.9.3-final.20260618.7z",
      "sha256": %q,
      "sizeBytes": 100
    },
    "reshade": {
      "standard": {
        "version": %q,
        "assetName": %q,
        "url": %q,
        "sha256": %q,
        "sizeBytes": %d
      },
      "addon": {
        "version": %q,
        "assetName": %q,
        "url": %q,
        "sha256": %q,
        "sizeBytes": %d
      }
    }
  }
}`, strings.Repeat("c", 64), standardVersion, standardAssetName, standardURL, standardDigest, standardSize, addonVersion, addonAssetName, addonURL, addonDigest, addonSize))
}

func TestAcquireInstallerCachesExactVersionsAndRevalidates(t *testing.T) {
	t.Parallel()

	downloads := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		downloads[request.URL.Path]++
		_, _ = fmt.Fprintf(writer, "installer:%s", request.URL.Path)
	}))
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	cacheDir := t.TempDir()
	signature := InstallerSignature{
		Status:     InstallerSignatureStatusVerified,
		Subject:    reShadeSignerSubject,
		SPKISHA256: reShadeSignerSPKISHA256,
	}
	verifier := signatureVerifierFunc(func(_ string, variant InstallerVariant) (InstallerSignature, error) {
		if variant != InstallerVariantStandard {
			t.Fatalf("variant = %q", variant)
		}
		return signature, nil
	})
	acquire := func(version string) InstallerArtifact {
		t.Helper()
		name := installerFileName(version, InstallerVariantStandard)
		artifact, err := AcquireInstaller(
			context.Background(),
			InstallerRelease{
				Version:   version,
				Variant:   InstallerVariantStandard,
				AssetName: name,
				URL:       server.URL + "/" + name,
			},
			InstallerAcquireOptions{
				CacheDir:             cacheDir,
				HTTPClient:           server.Client(),
				SignatureVerifier:    verifier,
				TrustedDownloadHosts: []string{serverURL.Hostname()},
				AllowHTTP:            true,
			},
		)
		if err != nil {
			t.Fatalf("AcquireInstaller(%q) error = %v", version, err)
		}
		return artifact
	}

	first := acquire("6.7.3")
	second := acquire("6.8.0")
	if downloads["/"+first.AssetName] != 1 || downloads["/"+second.AssetName] != 1 {
		t.Fatalf("downloads = %#v", downloads)
	}
	if _, err := os.Stat(first.Path); err != nil {
		t.Fatalf("older cached installer was removed: %v", err)
	}

	if err := os.WriteFile(second.Path, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	repaired := acquire("6.8.0")
	if downloads["/"+second.AssetName] != 2 {
		t.Fatalf("downloads = %#v, want corrupt cache redownload", downloads)
	}
	if repaired.SHA256 == second.SHA256 && repaired.SizeBytes == int64(len("corrupt")) {
		t.Fatalf("repaired artifact = %+v", repaired)
	}
	if _, err := os.Stat(second.Path + ".json"); err != nil {
		t.Fatalf("cache sidecar missing: %v", err)
	}
}

func TestAcquireInstallerRequiresBothAddonAcknowledgements(t *testing.T) {
	t.Parallel()

	release := InstallerRelease{
		Version:   "6.7.3",
		Variant:   InstallerVariantAddon,
		AssetName: "ReShade_Setup_6.7.3_Addon.exe",
		URL:       "https://reshade.me/downloads/ReShade_Setup_6.7.3_Addon.exe",
	}
	for _, acknowledgements := range []InstallerAcknowledgements{
		{},
		{SinglePlayerAcknowledged: true},
		{AntiCheatRiskAcknowledged: true},
	} {
		_, err := AcquireInstaller(context.Background(), release, InstallerAcquireOptions{
			CacheDir:         filepath.Join(t.TempDir(), "cache"),
			Acknowledgements: acknowledgements,
		})
		if err == nil || !strings.Contains(err.Error(), "separate single-player and anti-cheat") {
			t.Fatalf("AcquireInstaller() error = %v, want acknowledgement error", err)
		}
	}
}

func TestValidateInstallerReleaseRejectsPreviewApplySubstitution(t *testing.T) {
	t.Parallel()

	err := validateInstallerRelease(InstallerRelease{
		Version:   "6.7.3",
		Variant:   InstallerVariantStandard,
		AssetName: "ReShade_Setup_6.8.0.exe",
		URL:       "https://reshade.me/downloads/ReShade_Setup_6.8.0.exe",
	}, nil, false)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("validateInstallerRelease() error = %v", err)
	}
}
