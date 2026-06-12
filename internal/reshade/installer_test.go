package reshade

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadAndOpenInstallerDownloadsHighestStableTagAndOpens(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	writeReShadeTestFile(t, filepath.Join(cacheDir, "ReShade_Setup_6.7.3.exe"))
	addonPath := filepath.Join(cacheDir, "ReShade_Setup_6.7.3_Addon.exe")
	writeReShadeTestFile(t, addonPath)

	var downloadedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tags":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"name":"v6.7.3"},
				{"name":"v6.10.0"},
				{"name":"not-a-version"},
				{"name":"v6.10.1-beta"}
			]`))
		case "/downloads/ReShade_Setup_6.10.0.exe":
			downloadedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("installer"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var openedPath string
	result, err := DownloadAndOpenInstaller(context.Background(), InstallerOptions{
		TagsURL:         server.URL + "/tags",
		DownloadBaseURL: server.URL + "/downloads",
		CacheDir:        cacheDir,
		HTTPClient:      server.Client(),
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
		t.Fatalf("downloaded path = %q, want latest installer path", downloadedPath)
	}
	if filepath.Base(openedPath) != "ReShade_Setup_6.10.0.exe" {
		t.Fatalf("opened path = %q, want latest installer", openedPath)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "ReShade_Setup_6.7.3.exe")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old installer stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(addonPath); err != nil {
		t.Fatalf("add-on installer stat error = %v, want preserved", err)
	}
}

func TestDownloadAndOpenInstallerReusesCachedLatestInstaller(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	latestPath := filepath.Join(cacheDir, "ReShade_Setup_6.7.3.exe")
	writeReShadeTestFile(t, latestPath)

	var unexpectedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tags":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"name":"v6.7.3"}]`))
		default:
			unexpectedPath = r.URL.Path
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var openedPath string
	result, err := DownloadAndOpenInstaller(context.Background(), InstallerOptions{
		TagsURL:         server.URL + "/tags",
		DownloadBaseURL: server.URL + "/downloads",
		CacheDir:        cacheDir,
		HTTPClient:      server.Client(),
		OpenInstaller: func(_ context.Context, path string) error {
			openedPath = path
			return nil
		},
	})
	if err != nil {
		t.Fatalf("DownloadAndOpenInstaller() error = %v", err)
	}
	if unexpectedPath != "" {
		t.Fatalf("unexpected request path %q", unexpectedPath)
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

	var downloadedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tags":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"name":"v6.7.3"}]`))
		case "/downloads/ReShade_Setup_6.7.3_Addon.exe":
			downloadedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("addon installer"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var openedPath string
	result, err := DownloadAndOpenInstaller(context.Background(), InstallerOptions{
		TagsURL:         server.URL + "/tags",
		DownloadBaseURL: server.URL + "/downloads",
		CacheDir:        cacheDir,
		HTTPClient:      server.Client(),
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

	var unexpectedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tags":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"name":"v6.7.3"}]`))
		default:
			unexpectedPath = r.URL.Path
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var openedPath string
	_, err := DownloadAndOpenInstaller(context.Background(), InstallerOptions{
		TagsURL:         server.URL + "/tags",
		DownloadBaseURL: server.URL + "/downloads",
		CacheDir:        cacheDir,
		HTTPClient:      server.Client(),
		OpenInstaller: func(_ context.Context, path string) error {
			openedPath = path
			return nil
		},
		Variant: InstallerVariantAddon,
	})
	if err != nil {
		t.Fatalf("DownloadAndOpenInstaller() error = %v", err)
	}
	if unexpectedPath != "" {
		t.Fatalf("unexpected request path %q", unexpectedPath)
	}
	if openedPath != latestPath {
		t.Fatalf("opened path = %q, want cached add-on path %q", openedPath, latestPath)
	}
}

func TestDownloadAndOpenInstallerSurfacesFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		openError error
		want      string
	}{
		{
			name: "tag API status",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "unavailable", http.StatusBadGateway)
			},
			want: "fetch ReShade tags",
		},
		{
			name: "invalid tag JSON",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{`))
			},
			want: "decode ReShade tags response",
		},
		{
			name: "no stable tag",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`[{"name":"main"},{"name":"v6.7"}]`))
			},
			want: "no stable version tags found",
		},
		{
			name: "download status",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/tags" {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`[{"name":"v6.7.3"}]`))
					return
				}
				http.Error(w, "missing", http.StatusNotFound)
			},
			want: "download ReShade installer",
		},
		{
			name: "empty download",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if r.URL.Path == "/tags" {
					_, _ = w.Write([]byte(`[{"name":"v6.7.3"}]`))
				}
			},
			want: "empty response body",
		},
		{
			name: "open error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if r.URL.Path == "/tags" {
					_, _ = w.Write([]byte(`[{"name":"v6.7.3"}]`))
					return
				}
				_, _ = w.Write([]byte("installer"))
			},
			openError: errors.New("blocked"),
			want:      "open installer",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(tt.handler)
			defer server.Close()

			_, err := DownloadAndOpenInstaller(context.Background(), InstallerOptions{
				TagsURL:         server.URL + "/tags",
				DownloadBaseURL: server.URL + "/downloads",
				CacheDir:        t.TempDir(),
				HTTPClient:      server.Client(),
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := DownloadAndOpenInstaller(context.Background(), InstallerOptions{
		TagsURL:    server.URL + "/tags",
		CacheDir:   t.TempDir(),
		HTTPClient: server.Client(),
		Variant:    InstallerVariantAddon,
	})
	if err == nil {
		t.Fatal("DownloadAndOpenInstaller() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "prepare ReShade add-on installer") ||
		!strings.Contains(err.Error(), "fetch ReShade tags") {
		t.Fatalf("DownloadAndOpenInstaller() error = %q, want add-on helper and fetch context", err)
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
