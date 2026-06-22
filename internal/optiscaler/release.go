package optiscaler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/phergul/fiach/internal/thirdparty"
)

const DefaultReleasesURL = thirdparty.DefaultManifestURL

var finalAssetName = regexp.MustCompile(`(?i)^optiscaler_.*final.*\.7z$`)

type ReleaseOptions struct {
	ReleasesURL          string
	CacheDir             string
	HTTPClient           *http.Client
	ManifestJSON         []byte
	RefreshManifest      bool
	TrustedDownloadHosts []string
	AllowHTTP            bool
}

func DiscoverStableRelease(ctx context.Context, options ReleaseOptions) (release Release, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("discover stable OptiScaler release: %w", err)
		}
	}()

	options = normalizeReleaseOptions(options)
	result, err := thirdparty.Load(ctx, thirdparty.LoadOptions{
		Refresh:     options.RefreshManifest,
		ManifestURL: options.ReleasesURL,
		HTTPClient:  options.HTTPClient,
		Bundled:     options.ManifestJSON,
	})
	if err != nil {
		return Release{}, err
	}
	entry := result.Manifest.Tools.OptiScaler
	release = Release{
		Tag:       entry.Tag,
		Version:   entry.Version,
		AssetName: entry.AssetName,
		URL:       entry.URL,
		Digest:    entry.SHA256,
		Size:      entry.SizeBytes,
		Error:     result.Warning,
	}
	if err := validateRelease(release, nil, false); err != nil {
		return Release{}, err
	}
	return release, nil
}

func EnsureReleaseArchive(ctx context.Context, release Release, options ReleaseOptions) (path string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cache verified OptiScaler release archive: %w", err)
		}
	}()

	options = normalizeReleaseOptions(options)
	if err := validateRelease(release, options.TrustedDownloadHosts, options.AllowHTTP); err != nil {
		return "", err
	}
	cachePath := filepath.Join(options.CacheDir, safePathSegment(release.Tag), release.Digest, release.AssetName)
	if matches, err := archiveMatches(cachePath, release); err == nil && matches {
		return cachePath, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, release.URL, nil)
	if err != nil {
		return "", err
	}
	response, err := options.HTTPClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("download returned %s", response.Status)
	}
	temp, err := os.CreateTemp(filepath.Dir(cachePath), ".optiscaler-*.tmp")
	if err != nil {
		return "", err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(temp, hash), response.Body)
	closeErr := temp.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if written != release.Size || !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), release.Digest) {
		return "", errors.New("downloaded archive does not match the release digest and size")
	}
	if err := os.Remove(cachePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.Rename(tempPath, cachePath); err != nil {
		return "", err
	}
	return cachePath, nil
}

func normalizeReleaseOptions(options ReleaseOptions) ReleaseOptions {
	if options.ReleasesURL == "" {
		options.ReleasesURL = DefaultReleasesURL
	}
	if options.HTTPClient == nil {
		options.HTTPClient = http.DefaultClient
	}
	return options
}

func validateRelease(release Release, trustedHosts []string, allowHTTP bool) error {
	if release.Tag == "" || release.AssetName == "" || release.URL == "" || release.Digest == "" || release.Size <= 0 {
		return errors.New("release metadata is incomplete")
	}
	if !finalAssetName.MatchString(release.AssetName) {
		return errors.New("release asset does not match the final-build naming rule")
	}
	if filepath.Base(release.AssetName) != release.AssetName || strings.ContainsAny(release.AssetName, `/\`) {
		return errors.New("release asset name contains path separators")
	}
	if len(release.Digest) != sha256.Size*2 {
		return errors.New("release asset SHA-256 digest is malformed")
	}
	if _, err := hex.DecodeString(release.Digest); err != nil {
		return errors.New("release asset SHA-256 digest is malformed")
	}
	parsed, err := url.Parse(release.URL)
	if err != nil {
		return fmt.Errorf("parse release URL: %w", err)
	}
	if parsed.Scheme != "https" && !(allowHTTP && parsed.Scheme == "http") {
		return errors.New("release URL must use HTTPS")
	}
	if len(trustedHosts) == 0 {
		trustedHosts = []string{"github.com"}
	}
	trusted := false
	for _, host := range trustedHosts {
		if strings.EqualFold(strings.TrimSpace(host), parsed.Hostname()) {
			trusted = true
			break
		}
	}
	if parsed.User != nil || !trusted {
		return fmt.Errorf("release URL host %q is not trusted", parsed.Hostname())
	}
	if strings.EqualFold(parsed.Hostname(), "github.com") &&
		!strings.HasPrefix(parsed.EscapedPath(), "/optiscaler/OptiScaler/releases/download/") {
		return fmt.Errorf("release URL path %q is not trusted", parsed.EscapedPath())
	}
	return nil
}

func archiveMatches(path string, release Release) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return false, err
	}
	return size == release.Size && strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), release.Digest), nil
}

func safePathSegment(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	if value == "" || value == "." || value == ".." {
		return "release"
	}
	return value
}
