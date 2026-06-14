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

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/tags" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`[
			{"name":"v6.7.3"},
			{"name":"v6.10.0"},
			{"name":"v6.11.0-beta"},
			{"name":"main"}
		]`))
	}))
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	release, err := ResolveLatestInstaller(
		context.Background(),
		InstallerVariantAddon,
		InstallerResolveOptions{
			TagsURL:              server.URL + "/tags",
			DownloadBaseURL:      server.URL + "/downloads",
			HTTPClient:           server.Client(),
			TrustedDownloadHosts: []string{serverURL.Hostname()},
			AllowHTTP:            true,
		},
	)
	if err != nil {
		t.Fatalf("ResolveLatestInstaller() error = %v", err)
	}
	if release.Version != "6.10.0" ||
		release.AssetName != "ReShade_Setup_6.10.0_Addon.exe" ||
		release.URL != server.URL+"/downloads/ReShade_Setup_6.10.0_Addon.exe" {
		t.Fatalf("release = %+v", release)
	}
}

func TestResolveLatestInstallerRejectsUntrustedDownloadHost(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`[{"name":"v6.7.3"}]`))
	}))
	defer server.Close()

	_, err := ResolveLatestInstaller(
		context.Background(),
		InstallerVariantStandard,
		InstallerResolveOptions{
			TagsURL:         server.URL,
			DownloadBaseURL: server.URL,
			HTTPClient:      server.Client(),
			AllowHTTP:       true,
		},
	)
	if err == nil || !strings.Contains(err.Error(), "is not trusted") {
		t.Fatalf("ResolveLatestInstaller() error = %v, want untrusted host", err)
	}
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
