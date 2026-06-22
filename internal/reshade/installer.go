package reshade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/thirdparty"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	DefaultTagsURL         = thirdparty.DefaultManifestURL
	DefaultDownloadBaseURL = "https://reshade.me/downloads"
)

type OpenInstallerFunc func(ctx context.Context, path string) error

type InstallerVariant string

const (
	InstallerVariantStandard InstallerVariant = ""
	InstallerVariantAddon    InstallerVariant = "addon"
)

type InstallerOptions struct {
	TagsURL              string
	DownloadBaseURL      string
	CacheDir             string
	HTTPClient           *http.Client
	OpenInstaller        OpenInstallerFunc
	Variant              InstallerVariant
	ManifestJSON         []byte
	TrustedDownloadHosts []string
	AllowHTTP            bool
}

type InstallerLaunchResult struct {
	Version       string
	InstallerPath string
}

func DownloadAndOpenInstaller(ctx context.Context, opts InstallerOptions) (result InstallerLaunchResult, err error) {
	opts = normalizeInstallerOptions(opts)

	defer func() {
		if err != nil {
			err = fmt.Errorf("prepare %s: %w", installerDescription(opts.Variant), err)
		}
	}()

	release, err := ResolveLatestInstaller(ctx, opts.Variant, InstallerResolveOptions{
		TagsURL:              opts.TagsURL,
		DownloadBaseURL:      opts.DownloadBaseURL,
		HTTPClient:           opts.HTTPClient,
		TrustedDownloadHosts: opts.TrustedDownloadHosts,
		AllowHTTP:            opts.AllowHTTP,
		ManifestJSON:         opts.ManifestJSON,
	})
	if err != nil {
		return InstallerLaunchResult{}, err
	}

	installerPath, err := ensureInstaller(
		ctx,
		opts.HTTPClient,
		opts.CacheDir,
		release,
		opts.TrustedDownloadHosts,
		opts.AllowHTTP,
	)
	if err != nil {
		return InstallerLaunchResult{}, err
	}

	if err := removeStaleInstallers(opts.CacheDir, filepath.Base(installerPath), opts.Variant); err != nil {
		return InstallerLaunchResult{}, err
	}

	if err := opts.OpenInstaller(ctx, installerPath); err != nil {
		return InstallerLaunchResult{}, fmt.Errorf("open installer %q: %w", installerPath, err)
	}

	return InstallerLaunchResult{
		Version:       release.Version,
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
	return filepath.Join(application.Path(application.PathCacheHome), "fiach", "reshade", "installers")
}

func OpenInstaller(ctx context.Context, path string) error {
	command := exec.CommandContext(ctx, "cmd", "/C", "start", "", path)
	return command.Run()
}

func ensureInstaller(
	ctx context.Context,
	client *http.Client,
	cacheDir string,
	release InstallerRelease,
	trustedHosts []string,
	allowHTTP bool,
) (string, error) {
	if err := validateInstallerRelease(release, trustedHosts, allowHTTP); err != nil {
		return "", err
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create ReShade installer cache: %w", err)
	}

	name := release.AssetName
	path := filepath.Join(cacheDir, name)
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		if err := validateCachedInstallerFile(path, release); err == nil {
			return path, nil
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect cached ReShade installer: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, release.URL, nil)
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
	if response.Request == nil || response.Request.URL == nil {
		return "", errors.New("download ReShade installer: final response URL is missing")
	}
	if err := validateInstallerSource(response.Request.URL.String(), trustedHosts, allowHTTP); err != nil {
		return "", fmt.Errorf("validate final ReShade installer download URL: %w", err)
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
	if err := validateCachedInstallerFile(tempPath, release); err != nil {
		return "", err
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

func validateCachedInstallerFile(path string, release InstallerRelease) error {
	hash, size, err := fileops.FileIntegrity(path)
	if err != nil {
		return err
	}
	artifact := InstallerArtifact{
		InstallerRelease: release,
		Path:             path,
		SizeBytes:        size,
		SHA256:           hash,
	}
	return validateInstallerArtifactIntegrity(artifact, release)
}

func installerDownloadURL(downloadBaseURL string, installerVersion string, variant InstallerVariant) (string, error) {
	base, err := url.Parse(downloadBaseURL)
	if err != nil {
		return "", fmt.Errorf("parse ReShade download base URL: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("parse ReShade download base URL: %q is not absolute", downloadBaseURL)
	}

	base.Path = strings.TrimRight(base.Path, "/") + "/" + installerFileName(installerVersion, variant)
	return base.String(), nil
}

func installerFileName(installerVersion string, variant InstallerVariant) string {
	if variant == InstallerVariantAddon {
		return fmt.Sprintf("ReShade_Setup_%s_Addon.exe", installerVersion)
	}

	return fmt.Sprintf("ReShade_Setup_%s.exe", installerVersion)
}

func installerDescription(variant InstallerVariant) string {
	if variant == InstallerVariantAddon {
		return "ReShade add-on installer"
	}

	return "ReShade installer"
}

func removeStaleInstallers(cacheDir string, keepName string, variant InstallerVariant) error {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return fmt.Errorf("read ReShade installer cache: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == keepName || !isReShadeInstallerNameForVariant(entry.Name(), variant) {
			continue
		}
		if err := os.Remove(filepath.Join(cacheDir, entry.Name())); err != nil {
			return fmt.Errorf("remove stale ReShade installer %q: %w", entry.Name(), err)
		}
	}

	return nil
}

func isReShadeInstallerNameForVariant(name string, variant InstallerVariant) bool {
	if !strings.HasPrefix(name, "ReShade_Setup_") || !strings.HasSuffix(name, ".exe") {
		return false
	}

	isAddon := strings.HasSuffix(name, "_Addon.exe")
	return isAddon == (variant == InstallerVariantAddon)
}
