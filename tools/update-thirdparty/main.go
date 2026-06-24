package main

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
	"sort"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/phergul/fiach/internal/thirdparty"
)

const (
	optiscalerReleasesURL = "https://api.github.com/repos/optiscaler/OptiScaler/releases?per_page=30"
	reshadeTagsURL        = "https://api.github.com/repos/crosire/reshade/tags?per_page=100"
	reshadeDownloadBase   = "https://reshade.me/downloads"
	manifestPath          = "internal/thirdparty/releases.json"
)

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
	Size               int64  `json:"size"`
}

type tagResponse struct {
	Name string `json:"name"`
}

func main() {
	if err := run(context.Background()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "update third-party manifest: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("run updater: %w", err)
		}
	}()

	client := &http.Client{
		Timeout: 5 * time.Minute,
	}
	optiScaler, err := resolveOptiScaler(ctx, client)
	if err != nil {
		return err
	}
	standard, err := resolveReShade(ctx, client, false)
	if err != nil {
		return err
	}
	addon, err := resolveReShade(ctx, client, true)
	if err != nil {
		return err
	}
	manifest := thirdparty.Manifest{
		Version:   thirdparty.ManifestVersion,
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
		Tools: thirdparty.Tools{
			OptiScaler: optiScaler,
			ReShade: thirdparty.ReShadeReleases{
				Standard: standard,
				Addon:    addon,
			},
		},
	}
	if err := thirdparty.Validate(manifest); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	contents = append(contents, '\n')
	if err := os.WriteFile(manifestPath, contents, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", manifestPath, err)
	}
	return nil
}

func resolveOptiScaler(ctx context.Context, client *http.Client) (thirdparty.OptiScalerRelease, error) {
	var releases []githubRelease
	if err := getJSON(ctx, client, optiscalerReleasesURL, &releases); err != nil {
		return thirdparty.OptiScalerRelease{}, fmt.Errorf("fetch OptiScaler releases: %w", err)
	}
	sort.SliceStable(releases, func(i int, j int) bool {
		return releases[i].PublishedAt.After(releases[j].PublishedAt)
	})
	for _, release := range releases {
		if release.Draft || release.Prerelease {
			continue
		}
		matches := make([]githubAsset, 0, 1)
		for _, asset := range release.Assets {
			if thirdparty.IsOptiScalerFinalAsset(asset.Name) {
				matches = append(matches, asset)
			}
		}
		if len(matches) != 1 {
			return thirdparty.OptiScalerRelease{}, fmt.Errorf(
				"release %q has %d matching final archives",
				release.TagName,
				len(matches),
			)
		}
		hash, size, err := downloadIntegrity(ctx, client, matches[0].BrowserDownloadURL)
		if err != nil {
			return thirdparty.OptiScalerRelease{}, err
		}
		version := strings.TrimSpace(release.Name)
		if version == "" {
			version = release.TagName
		}
		return thirdparty.OptiScalerRelease{
			Tag:       release.TagName,
			Version:   version,
			AssetName: matches[0].Name,
			URL:       matches[0].BrowserDownloadURL,
			SHA256:    hash,
			SizeBytes: size,
		}, nil
	}
	return thirdparty.OptiScalerRelease{}, errors.New("no stable OptiScaler release found")
}

func resolveReShade(ctx context.Context, client *http.Client, addon bool) (thirdparty.ReShadeInstaller, error) {
	version, err := latestReShadeVersion(ctx, client)
	if err != nil {
		return thirdparty.ReShadeInstaller{}, err
	}
	assetName := fmt.Sprintf("ReShade_Setup_%s.exe", version)
	if addon {
		assetName = fmt.Sprintf("ReShade_Setup_%s_Addon.exe", version)
	}
	rawURL := strings.TrimRight(reshadeDownloadBase, "/") + "/" + assetName
	hash, size, err := downloadIntegrity(ctx, client, rawURL)
	if err != nil {
		return thirdparty.ReShadeInstaller{}, err
	}
	return thirdparty.ReShadeInstaller{
		Version:   version,
		AssetName: assetName,
		URL:       rawURL,
		SHA256:    hash,
		SizeBytes: size,
	}, nil
}

func latestReShadeVersion(ctx context.Context, client *http.Client) (string, error) {
	var tags []tagResponse
	if err := getJSON(ctx, client, reshadeTagsURL, &tags); err != nil {
		return "", fmt.Errorf("fetch ReShade tags: %w", err)
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
		return "", errors.New("no stable ReShade version tags found")
	}
	return strings.TrimPrefix(latest, "v"), nil
}

func getJSON(ctx context.Context, client *http.Client, rawURL string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "fiach-thirdparty-manifest-updater")
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("unexpected status %s", response.Status)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return err
	}
	return nil
}

func downloadIntegrity(ctx context.Context, client *http.Client, rawURL string) (string, int64, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", 0, err
	}
	request.Header.Set("User-Agent", "fiach-thirdparty-manifest-updater")
	response, err := client.Do(request)
	if err != nil {
		return "", 0, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", 0, fmt.Errorf("download %s: unexpected status %s", rawURL, response.Status)
	}
	temp, err := os.CreateTemp("", "fiach-thirdparty-*"+filepath.Ext(rawURL))
	if err != nil {
		return "", 0, err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(temp, hash), response.Body)
	closeErr := temp.Close()
	if copyErr != nil {
		return "", 0, copyErr
	}
	if closeErr != nil {
		return "", 0, closeErr
	}
	if size <= 0 {
		return "", 0, fmt.Errorf("download %s: empty response body", rawURL)
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}
