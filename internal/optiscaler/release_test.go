package optiscaler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverStableReleaseReturnsManifestRelease(t *testing.T) {
	t.Parallel()

	archive := []byte("archive")
	digest := sha256.Sum256(archive)
	release, err := DiscoverStableRelease(context.Background(), ReleaseOptions{
		ManifestJSON: optiScalerManifest(
			"Optiscaler_1.2.3a-final.7z",
			"https://github.com/optiscaler/OptiScaler/releases/download/v1.2.3/Optiscaler_1.2.3a-final.7z",
			hex.EncodeToString(digest[:]),
			int64(len(archive)),
		),
	})
	if err != nil {
		t.Fatalf("DiscoverStableRelease() error = %v", err)
	}
	if release.Tag != "v1.2.3" ||
		release.Version != "OptiScaler v1.2.3a" ||
		release.AssetName != "Optiscaler_1.2.3a-final.7z" ||
		release.Digest != hex.EncodeToString(digest[:]) ||
		release.Size != int64(len(archive)) {
		t.Fatalf("release = %+v", release)
	}
}

func TestEnsureReleaseArchiveCachesVerifiedManifestArchive(t *testing.T) {
	t.Parallel()

	archive := []byte("archive")
	digest := sha256.Sum256(archive)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(archive)
	}))
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	release := Release{
		Tag:       "v1.2.3",
		Version:   "OptiScaler v1.2.3a",
		AssetName: "Optiscaler_1.2.3a-final.7z",
		URL:       server.URL + "/archive",
		Digest:    hex.EncodeToString(digest[:]),
		Size:      int64(len(archive)),
	}
	path, err := EnsureReleaseArchive(context.Background(), release, ReleaseOptions{
		CacheDir:             t.TempDir(),
		HTTPClient:           server.Client(),
		TrustedDownloadHosts: []string{serverURL.Hostname()},
		AllowHTTP:            true,
	})
	if err != nil {
		t.Fatalf("EnsureReleaseArchive() error = %v", err)
	}
	if contents, err := os.ReadFile(path); err != nil || string(contents) != string(archive) {
		t.Fatalf("cached archive = %q, %v", contents, err)
	}
}

func TestDiscoverStableReleaseRejectsInvalidManifestDigest(t *testing.T) {
	t.Parallel()

	_, err := DiscoverStableRelease(context.Background(), ReleaseOptions{
		ManifestJSON: optiScalerManifest(
			"Optiscaler_1.2.3a-final.7z",
			"https://github.com/optiscaler/OptiScaler/releases/download/v1.2.3/Optiscaler_1.2.3a-final.7z",
			"abcd",
			10,
		),
	})
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("DiscoverStableRelease() error = %v, want SHA-256 error", err)
	}
}

func TestCleanPackageEntryRejectsTraversalAndAbsolutePaths(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"../evil.dll", "/evil.dll", `C:\evil.dll`} {
		if _, err := cleanPackageEntry(value); err == nil {
			t.Fatalf("cleanPackageEntry(%q) error = nil", value)
		}
	}
}

func TestEnsureReleaseArchiveRejectsDigestMismatch(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("wrong"))
	}))
	defer server.Close()
	release := Release{
		Tag:       "v1",
		Version:   "v1",
		AssetName: "Optiscaler_v1-final.7z",
		URL:       server.URL,
		Digest:    strings.Repeat("a", 64),
		Size:      5,
	}
	_, err := EnsureReleaseArchive(context.Background(), release, ReleaseOptions{
		CacheDir:             filepath.Join(t.TempDir(), "cache"),
		HTTPClient:           server.Client(),
		TrustedDownloadHosts: []string{"127.0.0.1", "::1"},
		AllowHTTP:            true,
	})
	if err == nil {
		t.Fatal("EnsureReleaseArchive() error = nil, want digest mismatch")
	}
}

func optiScalerManifest(assetName string, rawURL string, digest string, size int64) []byte {
	return []byte(fmt.Sprintf(`{
  "version": 1,
  "updatedAt": "2026-06-22T00:00:00Z",
  "tools": {
    "optiscaler": {
      "tag": "v1.2.3",
      "version": "OptiScaler v1.2.3a",
      "assetName": %q,
      "url": %q,
      "sha256": %q,
      "sizeBytes": %d
    },
    "reshade": {
      "standard": {
        "version": "6.7.3",
        "assetName": "ReShade_Setup_6.7.3.exe",
        "url": "https://reshade.me/downloads/ReShade_Setup_6.7.3.exe",
        "sha256": %q,
        "sizeBytes": 100
      },
      "addon": {
        "version": "6.7.3",
        "assetName": "ReShade_Setup_6.7.3_Addon.exe",
        "url": "https://reshade.me/downloads/ReShade_Setup_6.7.3_Addon.exe",
        "sha256": %q,
        "sizeBytes": 100
      }
    }
  }
}`, assetName, rawURL, digest, size, strings.Repeat("b", 64), strings.Repeat("c", 64)))
}
