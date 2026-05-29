package reshade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	DefaultTagsURL         = "https://api.github.com/repos/crosire/reshade/tags?per_page=100"
	DefaultDownloadBaseURL = "https://reshade.me/downloads"
)

type OpenInstallerFunc func(ctx context.Context, path string) error

type InstallerOptions struct {
	TagsURL         string
	DownloadBaseURL string
	CacheDir        string
	HTTPClient      *http.Client
	OpenInstaller   OpenInstallerFunc
}

type InstallerLaunchResult struct {
	Version       string
	InstallerPath string
}

type tagResponse struct {
	Name string `json:"name"`
}

func DownloadAndOpenInstaller(ctx context.Context, opts InstallerOptions) (result InstallerLaunchResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("prepare ReShade installer: %w", err)
		}
	}()

	opts = normalizeInstallerOptions(opts)

	latestVersion, err := latestVersion(ctx, opts.HTTPClient, opts.TagsURL)
	if err != nil {
		return InstallerLaunchResult{}, err
	}

	installerPath, err := ensureInstaller(ctx, opts.HTTPClient, opts.DownloadBaseURL, opts.CacheDir, latestVersion)
	if err != nil {
		return InstallerLaunchResult{}, err
	}

	if err := removeStaleInstallers(opts.CacheDir, filepath.Base(installerPath)); err != nil {
		return InstallerLaunchResult{}, err
	}

	if err := opts.OpenInstaller(ctx, installerPath); err != nil {
		return InstallerLaunchResult{}, fmt.Errorf("open installer %q: %w", installerPath, err)
	}

	return InstallerLaunchResult{
		Version:       latestVersion,
		InstallerPath: installerPath,
	}, nil
}

func normalizeInstallerOptions(opts InstallerOptions) InstallerOptions {
	if opts.TagsURL == "" {
		opts.TagsURL = DefaultTagsURL
	}
	if opts.DownloadBaseURL == "" {
		opts.DownloadBaseURL = DefaultDownloadBaseURL
	}
	if opts.CacheDir == "" {
		opts.CacheDir = DefaultInstallerCacheDir()
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}
	if opts.OpenInstaller == nil {
		opts.OpenInstaller = OpenInstaller
	}

	return opts
}

func DefaultInstallerCacheDir() string {
	return filepath.Join(application.Path(application.PathCacheHome), "mod-manager", "reshade", "installers")
}

func OpenInstaller(ctx context.Context, path string) error {
	command := exec.CommandContext(ctx, "cmd", "/C", "start", "", path)
	return command.Run()
}

func latestVersion(ctx context.Context, client *http.Client, tagsURL string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, tagsURL, nil)
	if err != nil {
		return "", fmt.Errorf("build ReShade tags request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("fetch ReShade tags: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("fetch ReShade tags: unexpected status %s", response.Status)
	}

	var tags []tagResponse
	if err := json.NewDecoder(response.Body).Decode(&tags); err != nil {
		return "", fmt.Errorf("decode ReShade tags response: %w", err)
	}

	latest := ""
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		if !semver.IsValid(name) || semver.Canonical(name) != name || semver.Prerelease(name) != "" {
			continue
		}
		if latest == "" || semver.Compare(name, latest) > 0 {
			latest = name
		}
	}
	if latest == "" {
		return "", errors.New("find latest ReShade version: no stable version tags found")
	}

	return strings.TrimPrefix(latest, "v"), nil
}

func ensureInstaller(ctx context.Context, client *http.Client, downloadBaseURL string, cacheDir string, installerVersion string) (string, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create ReShade installer cache: %w", err)
	}

	name := installerFileName(installerVersion)
	path := filepath.Join(cacheDir, name)
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return path, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect cached ReShade installer: %w", err)
	}

	downloadURL, err := installerDownloadURL(downloadBaseURL, installerVersion)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("build ReShade installer request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("download ReShade installer: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("download ReShade installer: unexpected status %s", response.Status)
	}

	tempFile, err := os.CreateTemp(cacheDir, name+".*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temporary ReShade installer: %w", err)
	}
	tempPath := tempFile.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()

	written, copyErr := io.Copy(tempFile, response.Body)
	closeErr := tempFile.Close()
	if copyErr != nil {
		return "", fmt.Errorf("write ReShade installer: %w", copyErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close ReShade installer: %w", closeErr)
	}
	if written == 0 {
		return "", errors.New("download ReShade installer: empty response body")
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("replace cached ReShade installer: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return "", fmt.Errorf("cache ReShade installer: %w", err)
	}
	removeTemp = false

	return path, nil
}

func installerDownloadURL(downloadBaseURL string, installerVersion string) (string, error) {
	base, err := url.Parse(downloadBaseURL)
	if err != nil {
		return "", fmt.Errorf("parse ReShade download base URL: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("parse ReShade download base URL: %q is not absolute", downloadBaseURL)
	}

	base.Path = strings.TrimRight(base.Path, "/") + "/" + installerFileName(installerVersion)
	return base.String(), nil
}

func installerFileName(installerVersion string) string {
	return fmt.Sprintf("ReShade_Setup_%s.exe", installerVersion)
}

func removeStaleInstallers(cacheDir string, keepName string) error {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return fmt.Errorf("read ReShade installer cache: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == keepName || !isReShadeInstallerName(entry.Name()) {
			continue
		}
		if err := os.Remove(filepath.Join(cacheDir, entry.Name())); err != nil {
			return fmt.Errorf("remove stale ReShade installer %q: %w", entry.Name(), err)
		}
	}

	return nil
}

func isReShadeInstallerName(name string) bool {
	return strings.HasPrefix(name, "ReShade_Setup_") && strings.HasSuffix(name, ".exe")
}
