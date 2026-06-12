package optiscaler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverStableReleaseSelectsOneDigestBearingFinalArchive(t *testing.T) {
	t.Parallel()

	archive := []byte("archive")
	digest := sha256.Sum256(archive)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/releases" {
			fmt.Fprintf(writer, `[{
				"tag_name":"v1.2.3","name":"OptiScaler v1.2.3a","draft":false,"prerelease":false,
				"published_at":"2026-06-01T00:00:00Z",
				"assets":[{"name":"Optiscaler_1.2.3a-final.7z","browser_download_url":"%s/archive","digest":"sha256:%s","size":%d}]
			}]`, "http://example.invalid", hex.EncodeToString(digest[:]), len(archive))
			return
		}
		_, _ = writer.Write(archive)
	}))
	defer server.Close()

	release, err := DiscoverStableRelease(context.Background(), ReleaseOptions{
		ReleasesURL: server.URL + "/releases", HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("DiscoverStableRelease() error = %v", err)
	}
	release.URL = server.URL + "/archive"
	path, err := EnsureReleaseArchive(context.Background(), release, ReleaseOptions{
		CacheDir: t.TempDir(), HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("EnsureReleaseArchive() error = %v", err)
	}
	if contents, err := os.ReadFile(path); err != nil || string(contents) != string(archive) {
		t.Fatalf("cached archive = %q, %v", contents, err)
	}
}

func TestDiscoverStableReleaseFailsClosedOnAmbiguousAssets(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(writer, `[{"tag_name":"v1","name":"v1","draft":false,"prerelease":false,
			"published_at":"2026-06-01T00:00:00Z","assets":[
			{"name":"Optiscaler_a-final.7z","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			{"name":"Optiscaler_b-final.7z","digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]}]`)
	}))
	defer server.Close()
	_, err := DiscoverStableRelease(context.Background(), ReleaseOptions{ReleasesURL: server.URL, HTTPClient: server.Client()})
	if err == nil || !strings.Contains(err.Error(), "2 matching") {
		t.Fatalf("DiscoverStableRelease() error = %v, want ambiguous asset error", err)
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
		Tag: "v1", Version: "v1", AssetName: "Optiscaler_v1-final.7z",
		URL: server.URL, Digest: strings.Repeat("a", 64), Size: 5,
	}
	_, err := EnsureReleaseArchive(context.Background(), release, ReleaseOptions{CacheDir: filepath.Join(t.TempDir(), "cache"), HTTPClient: server.Client()})
	if err == nil {
		t.Fatal("EnsureReleaseArchive() error = nil, want digest mismatch")
	}
}
