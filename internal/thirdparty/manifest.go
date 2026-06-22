package thirdparty

import (
	"context"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const (
	ManifestVersion    = 1
	DefaultManifestURL = "https://raw.githubusercontent.com/phergul/fiach/main/internal/thirdparty/releases.json"
)

const (
	optiscalerDownloadHost = "github.com"
	optiscalerDownloadPath = "/optiscaler/OptiScaler/releases/download/"
	reshadeDownloadHost    = "reshade.me"
	reshadeDownloadPath    = "/downloads/"
)

//go:embed releases.json
var manifestFS embed.FS

type Manifest struct {
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
	Tools     Tools     `json:"tools"`
}

type Tools struct {
	OptiScaler OptiScalerRelease `json:"optiscaler"`
	ReShade    ReShadeReleases   `json:"reshade"`
}

type OptiScalerRelease struct {
	Tag       string `json:"tag"`
	Version   string `json:"version"`
	AssetName string `json:"assetName"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"sizeBytes"`
}

type ReShadeReleases struct {
	Standard ReShadeInstaller `json:"standard"`
	Addon    ReShadeInstaller `json:"addon"`
}

type ReShadeInstaller struct {
	Version   string `json:"version"`
	AssetName string `json:"assetName"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"sizeBytes"`
}

type LoadOptions struct {
	Refresh     bool
	ManifestURL string
	HTTPClient  *http.Client
	Timeout     time.Duration
	Bundled     []byte
}

type LoadResult struct {
	Manifest Manifest
	Source   string
	Warning  string
}

func Load(ctx context.Context, options LoadOptions) (result LoadResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("load third-party release manifest: %w", err)
		}
	}()

	bundled, err := parseBundled(options.Bundled)
	if err != nil {
		return LoadResult{}, err
	}
	result = LoadResult{
		Manifest: bundled,
		Source:   "bundled",
	}
	if !options.Refresh {
		return result, nil
	}
	refreshed, err := fetchRefreshed(ctx, options)
	if err != nil {
		result.Warning = fmt.Sprintf("refresh third-party release manifest: %v", err)
		return result, nil
	}
	return LoadResult{
		Manifest: refreshed,
		Source:   "remote",
	}, nil
}

func Parse(contents []byte) (manifest Manifest, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("parse third-party release manifest: %w", err)
		}
	}()
	if err := json.Unmarshal(contents, &manifest); err != nil {
		return Manifest{}, err
	}
	if err := Validate(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func Validate(manifest Manifest) error {
	if manifest.Version != ManifestVersion {
		return fmt.Errorf("unsupported schema version %d", manifest.Version)
	}
	if manifest.UpdatedAt.IsZero() {
		return errors.New("updatedAt is required")
	}
	if err := validateOptiScaler(manifest.Tools.OptiScaler); err != nil {
		return err
	}
	if err := validateReShade("standard", manifest.Tools.ReShade.Standard); err != nil {
		return err
	}
	if err := validateReShade("addon", manifest.Tools.ReShade.Addon); err != nil {
		return err
	}
	return nil
}

func parseBundled(override []byte) (Manifest, error) {
	contents := override
	if len(contents) == 0 {
		var err error
		contents, err = manifestFS.ReadFile("releases.json")
		if err != nil {
			return Manifest{}, fmt.Errorf("read bundled manifest: %w", err)
		}
	}
	return Parse(contents)
}

func fetchRefreshed(ctx context.Context, options LoadOptions) (Manifest, error) {
	manifestURL := strings.TrimSpace(options.ManifestURL)
	if manifestURL == "" {
		manifestURL = DefaultManifestURL
	}
	if err := validateRawManifestURL(manifestURL); err != nil {
		return Manifest{}, err
	}
	client := options.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return Manifest{}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return Manifest{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return Manifest{}, fmt.Errorf("unexpected status %s", response.Status)
	}
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return Manifest{}, err
	}
	return Parse(contents)
}

func validateOptiScaler(release OptiScalerRelease) error {
	if strings.TrimSpace(release.Tag) == "" {
		return errors.New("OptiScaler tag is required")
	}
	if strings.TrimSpace(release.Version) == "" {
		return errors.New("OptiScaler version is required")
	}
	if err := validateArtifact("OptiScaler", release.AssetName, release.URL, release.SHA256, release.SizeBytes); err != nil {
		return err
	}
	return validateURLPrefix("OptiScaler", release.URL, optiscalerDownloadHost, optiscalerDownloadPath)
}

func validateReShade(name string, installer ReShadeInstaller) error {
	if strings.TrimSpace(installer.Version) == "" {
		return fmt.Errorf("ReShade %s version is required", name)
	}
	if err := validateArtifact("ReShade "+name, installer.AssetName, installer.URL, installer.SHA256, installer.SizeBytes); err != nil {
		return err
	}
	return validateURLPrefix("ReShade "+name, installer.URL, reshadeDownloadHost, reshadeDownloadPath)
}

func validateArtifact(label string, assetName string, rawURL string, digest string, size int64) error {
	if strings.TrimSpace(assetName) == "" {
		return fmt.Errorf("%s asset name is required", label)
	}
	if path.Base(assetName) != assetName || strings.ContainsAny(assetName, `/\`) {
		return fmt.Errorf("%s asset name contains path separators", label)
	}
	if strings.TrimSpace(rawURL) == "" {
		return fmt.Errorf("%s URL is required", label)
	}
	if !validSHA256(digest) {
		return fmt.Errorf("%s SHA-256 digest is malformed", label)
	}
	if size <= 0 {
		return fmt.Errorf("%s size must be positive", label)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse %s URL: %w", label, err)
	}
	if path.Base(parsed.Path) != assetName {
		return fmt.Errorf("%s URL path does not end with asset %q", label, assetName)
	}
	return nil
}

func validateURLPrefix(label string, rawURL string, host string, prefix string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse %s URL: %w", label, err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("%s URL must use HTTPS", label)
	}
	if parsed.User != nil || !strings.EqualFold(parsed.Hostname(), host) {
		return fmt.Errorf("%s URL host %q is not trusted", label, parsed.Hostname())
	}
	if !strings.HasPrefix(parsed.EscapedPath(), prefix) {
		return fmt.Errorf("%s URL path %q is not trusted", label, parsed.EscapedPath())
	}
	return nil
}

func validateRawManifestURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse manifest URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return errors.New("manifest URL must use HTTPS")
	}
	if parsed.User != nil || !strings.EqualFold(parsed.Hostname(), "raw.githubusercontent.com") {
		return fmt.Errorf("manifest URL host %q is not trusted", parsed.Hostname())
	}
	return nil
}

func validSHA256(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}
