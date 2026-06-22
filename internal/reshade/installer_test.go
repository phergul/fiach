package reshade

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestDownloadAndOpenInstallerDownloadsManifestReleaseAndOpens(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	writeReShadeTestFile(t, filepath.Join(cacheDir, "ReShade_Setup_6.7.3.exe"))
	addonPath := filepath.Join(cacheDir, "ReShade_Setup_6.7.3_Addon.exe")
	writeReShadeTestFile(t, addonPath)

	body := []byte("installer")
	var downloadedPath string
	client := testInstallerHTTPClient(func(request *http.Request) ([]byte, int) {
		downloadedPath = request.URL.Path
		return body, http.StatusOK
	})

	var openedPath string
	result, err := DownloadAndOpenInstaller(context.Background(), InstallerOptions{
		CacheDir:     cacheDir,
		HTTPClient:   client,
		ManifestJSON: installerManifest("6.10.0", body, []byte("addon")),
		OpenInstaller: func(_ context.Context, path string) error {
			openedPath = path
			return nil
		},
	})
	if err != nil {
		t.Fatalf("DownloadAndOpenInstaller() error = %v", err)
	}
	if result.Version != "6.10.0" {
		t.Fatalf("Version = %q, want 6.10.0", result.Version)
	}
	if downloadedPath != "/downloads/ReShade_Setup_6.10.0.exe" {
		t.Fatalf("downloaded path = %q, want manifest installer path", downloadedPath)
	}
	if filepath.Base(openedPath) != "ReShade_Setup_6.10.0.exe" {
		t.Fatalf("opened path = %q, want manifest installer", openedPath)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "ReShade_Setup_6.7.3.exe")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old installer stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(addonPath); err != nil {
		t.Fatalf("add-on installer stat error = %v, want preserved", err)
	}
}

func TestDownloadAndOpenInstallerReusesCachedManifestInstaller(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	latestPath := filepath.Join(cacheDir, "ReShade_Setup_6.7.3.exe")
	writeReShadeTestFile(t, latestPath)

	client := testInstallerHTTPClient(func(request *http.Request) ([]byte, int) {
		t.Fatalf("unexpected request path %q", request.URL.Path)
		return nil, http.StatusInternalServerError
	})

	var openedPath string
	result, err := DownloadAndOpenInstaller(context.Background(), InstallerOptions{
		CacheDir:     cacheDir,
		HTTPClient:   client,
		ManifestJSON: installerManifest("6.7.3", []byte("x"), []byte("addon")),
		OpenInstaller: func(_ context.Context, path string) error {
			openedPath = path
			return nil
		},
	})
	if err != nil {
		t.Fatalf("DownloadAndOpenInstaller() error = %v", err)
	}
	if result.Version != "6.7.3" {
		t.Fatalf("Version = %q, want 6.7.3", result.Version)
	}
	if openedPath != latestPath {
		t.Fatalf("opened path = %q, want cached latest path %q", openedPath, latestPath)
	}
}

func TestDownloadAndOpenInstallerDownloadsAddonVariant(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	standardPath := filepath.Join(cacheDir, "ReShade_Setup_6.7.3.exe")
	oldAddonPath := filepath.Join(cacheDir, "ReShade_Setup_6.6.0_Addon.exe")
	writeReShadeTestFile(t, standardPath)
	writeReShadeTestFile(t, oldAddonPath)

	body := []byte("addon installer")
	var downloadedPath string
	client := testInstallerHTTPClient(func(request *http.Request) ([]byte, int) {
		downloadedPath = request.URL.Path
		return body, http.StatusOK
	})

	var openedPath string
	result, err := DownloadAndOpenInstaller(context.Background(), InstallerOptions{
		CacheDir:     cacheDir,
		HTTPClient:   client,
		ManifestJSON: installerManifest("6.7.3", []byte("standard"), body),
		OpenInstaller: func(_ context.Context, path string) error {
			openedPath = path
			return nil
		},
		Variant: InstallerVariantAddon,
	})
	if err != nil {
		t.Fatalf("DownloadAndOpenInstaller() error = %v", err)
	}
	if result.Version != "6.7.3" {
		t.Fatalf("Version = %q, want 6.7.3", result.Version)
	}
	if downloadedPath != "/downloads/ReShade_Setup_6.7.3_Addon.exe" {
		t.Fatalf("downloaded path = %q, want add-on installer path", downloadedPath)
	}
	if filepath.Base(openedPath) != "ReShade_Setup_6.7.3_Addon.exe" {
		t.Fatalf("opened path = %q, want add-on installer", openedPath)
	}
	if _, err := os.Stat(standardPath); err != nil {
		t.Fatalf("standard installer stat error = %v, want preserved", err)
	}
	if _, err := os.Stat(oldAddonPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old add-on installer stat error = %v, want not exist", err)
	}
}

func TestDownloadAndOpenInstallerReusesCachedAddonVariant(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	latestPath := filepath.Join(cacheDir, "ReShade_Setup_6.7.3_Addon.exe")
	writeReShadeTestFile(t, latestPath)

	client := testInstallerHTTPClient(func(request *http.Request) ([]byte, int) {
		t.Fatalf("unexpected request path %q", request.URL.Path)
		return nil, http.StatusInternalServerError
	})

	var openedPath string
	_, err := DownloadAndOpenInstaller(context.Background(), InstallerOptions{
		CacheDir:     cacheDir,
		HTTPClient:   client,
		ManifestJSON: installerManifest("6.7.3", []byte("standard"), []byte("x")),
		OpenInstaller: func(_ context.Context, path string) error {
			openedPath = path
			return nil
		},
		Variant: InstallerVariantAddon,
	})
	if err != nil {
		t.Fatalf("DownloadAndOpenInstaller() error = %v", err)
	}
	if openedPath != latestPath {
		t.Fatalf("opened path = %q, want cached add-on path %q", openedPath, latestPath)
	}
}

func TestDownloadAndOpenInstallerSurfacesFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		manifestJSON []byte
		body         []byte
		status       int
		openError    error
		want         string
	}{
		{
			name:         "invalid manifest JSON",
			manifestJSON: []byte(`{`),
			want:         "parse third-party release manifest",
		},
		{
			name:         "download status",
			manifestJSON: installerManifest("6.7.3", []byte("installer"), []byte("addon")),
			status:       http.StatusNotFound,
			want:         "download ReShade installer",
		},
		{
			name:         "empty download",
			manifestJSON: installerManifest("6.7.3", []byte("installer"), []byte("addon")),
			status:       http.StatusOK,
			want:         "empty response body",
		},
		{
			name:         "digest mismatch",
			manifestJSON: installerManifest("6.7.3", []byte("installer"), []byte("addon")),
			body:         []byte("wrong"),
			status:       http.StatusOK,
			want:         "does not match the release size",
		},
		{
			name:         "open error",
			manifestJSON: installerManifest("6.7.3", []byte("installer"), []byte("addon")),
			body:         []byte("installer"),
			status:       http.StatusOK,
			openError:    errors.New("blocked"),
			want:         "open installer",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := testInstallerHTTPClient(func(*http.Request) ([]byte, int) {
				return tt.body, tt.status
			})
			_, err := DownloadAndOpenInstaller(context.Background(), InstallerOptions{
				CacheDir:     t.TempDir(),
				HTTPClient:   client,
				ManifestJSON: tt.manifestJSON,
				OpenInstaller: func(_ context.Context, _ string) error {
					return tt.openError
				},
			})
			if err == nil {
				t.Fatal("DownloadAndOpenInstaller() error = nil, want error")
			}
			if !strings.Contains(err.Error(), "prepare ReShade installer") || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("DownloadAndOpenInstaller() error = %q, want helper context and %q", err.Error(), tt.want)
			}
		})
	}
}

func TestDownloadAndOpenInstallerAddonFailureUsesVariantContext(t *testing.T) {
	t.Parallel()

	_, err := DownloadAndOpenInstaller(context.Background(), InstallerOptions{
		CacheDir:     t.TempDir(),
		ManifestJSON: []byte(`{`),
		Variant:      InstallerVariantAddon,
	})
	if err == nil {
		t.Fatal("DownloadAndOpenInstaller() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "prepare ReShade add-on installer") ||
		!strings.Contains(err.Error(), "parse third-party release manifest") {
		t.Fatalf("DownloadAndOpenInstaller() error = %q, want add-on helper and manifest context", err)
	}
}

func TestInstallerDownloadURLRejectsRelativeBase(t *testing.T) {
	t.Parallel()

	_, err := installerDownloadURL("/downloads", "6.7.3", InstallerVariantStandard)
	if err == nil {
		t.Fatal("installerDownloadURL() error = nil, want error")
	}
	if !strings.Contains(fmt.Sprint(err), "not absolute") {
		t.Fatalf("installerDownloadURL() error = %q, want not absolute", err)
	}
}

func installerManifest(version string, standardBody []byte, addonBody []byte) []byte {
	standardDigest, standardSize := testDigest(standardBody)
	addonDigest, addonSize := testDigest(addonBody)
	return reShadeManifest(
		version,
		fmt.Sprintf("ReShade_Setup_%s.exe", version),
		fmt.Sprintf("https://reshade.me/downloads/ReShade_Setup_%s.exe", version),
		standardDigest,
		standardSize,
		version,
		fmt.Sprintf("ReShade_Setup_%s_Addon.exe", version),
		fmt.Sprintf("https://reshade.me/downloads/ReShade_Setup_%s_Addon.exe", version),
		addonDigest,
		addonSize,
	)
}

func testInstallerHTTPClient(respond func(*http.Request) ([]byte, int)) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			body, status := respond(request)
			if status == 0 {
				status = http.StatusOK
			}
			return &http.Response{
				StatusCode: status,
				Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader(body)),
				Request:    request,
			}, nil
		}),
	}
}

func testDigest(contents []byte) (string, int64) {
	sum := sha256.Sum256(contents)
	return hex.EncodeToString(sum[:]), int64(len(contents))
}
