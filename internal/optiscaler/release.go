package optiscaler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const DefaultReleasesURL = "https://api.github.com/repos/optiscaler/OptiScaler/releases?per_page=30"

var finalAssetName = regexp.MustCompile(`(?i)^optiscaler_.*final.*\.7z$`)

type ReleaseOptions struct {
	ReleasesURL string
	CacheDir    string
	HTTPClient  *http.Client
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
	Size               int64  `json:"size"`
}

func DiscoverStableRelease(ctx context.Context, options ReleaseOptions) (release Release, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("discover stable OptiScaler release: %w", err)
		}
	}()

	options = normalizeReleaseOptions(options)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, options.ReleasesURL, nil)
	if err != nil {
		return Release{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	response, err := options.HTTPClient.Do(request)
	if err != nil {
		return Release{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Release{}, fmt.Errorf("GitHub returned %s", response.Status)
	}
	var releases []githubRelease
	if err := json.NewDecoder(response.Body).Decode(&releases); err != nil {
		return Release{}, err
	}
	sort.SliceStable(releases, func(i, j int) bool {
		return releases[i].PublishedAt.After(releases[j].PublishedAt)
	})
	for _, candidate := range releases {
		if candidate.Draft || candidate.Prerelease {
			continue
		}
		matches := make([]githubAsset, 0, 1)
		for _, asset := range candidate.Assets {
			if finalAssetName.MatchString(asset.Name) && strings.HasPrefix(strings.ToLower(asset.Digest), "sha256:") {
				matches = append(matches, asset)
			}
		}
		if len(matches) != 1 {
			return Release{}, fmt.Errorf("release %q has %d matching digest-bearing final archives", candidate.TagName, len(matches))
		}
		digest := strings.TrimPrefix(strings.ToLower(matches[0].Digest), "sha256:")
		if len(digest) != sha256.Size*2 {
			return Release{}, errors.New("release asset SHA-256 digest is malformed")
		}
		version := strings.TrimSpace(candidate.Name)
		if version == "" {
			version = candidate.TagName
		}
		return Release{
			Tag:       candidate.TagName,
			Version:   version,
			AssetName: matches[0].Name,
			URL:       matches[0].BrowserDownloadURL,
			Digest:    digest,
			Size:      matches[0].Size,
		}, nil
	}
	return Release{}, errors.New("no stable release was found")
}

func EnsureReleaseArchive(ctx context.Context, release Release, options ReleaseOptions) (path string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cache verified OptiScaler release archive: %w", err)
		}
	}()

	options = normalizeReleaseOptions(options)
	if err := validateRelease(release); err != nil {
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

func validateRelease(release Release) error {
	if release.Tag == "" || release.AssetName == "" || release.URL == "" || release.Digest == "" || release.Size <= 0 {
		return errors.New("release metadata is incomplete")
	}
	if !finalAssetName.MatchString(release.AssetName) {
		return errors.New("release asset does not match the final-build naming rule")
	}
	if filepath.Base(release.AssetName) != release.AssetName || strings.ContainsAny(release.AssetName, `/\`) {
		return errors.New("release asset name contains path separators")
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
