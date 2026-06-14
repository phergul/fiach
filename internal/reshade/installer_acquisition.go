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
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/phergul/fiach/internal/fileops"
)

const installerCacheMetadataVersion = 1

var installerCacheMu sync.Mutex

type InstallerSignatureStatus string

const (
	InstallerSignatureStatusVerified InstallerSignatureStatus = "verified"
	InstallerSignatureStatusUnsigned InstallerSignatureStatus = "unsigned"
)

type InstallerRelease struct {
	Version   string           `json:"version"`
	Variant   InstallerVariant `json:"variant"`
	AssetName string           `json:"assetName"`
	URL       string           `json:"url"`
}

type InstallerSignature struct {
	Status          InstallerSignatureStatus `json:"status"`
	Subject         string                   `json:"subject,omitempty"`
	SPKISHA256      string                   `json:"spkiSha256,omitempty"`
	CertificateSHA1 string                   `json:"certificateSha1,omitempty"`
}

type InstallerArtifact struct {
	InstallerRelease
	Path      string             `json:"path"`
	SizeBytes int64              `json:"sizeBytes"`
	SHA256    string             `json:"sha256"`
	Signature InstallerSignature `json:"signature"`
}

type InstallerAcknowledgements struct {
	SinglePlayerAcknowledged  bool
	AntiCheatRiskAcknowledged bool
}

type InstallerSignatureVerifier interface {
	VerifyInstallerSignature(string, InstallerVariant) (InstallerSignature, error)
}

type InstallerResolveOptions struct {
	TagsURL              string
	DownloadBaseURL      string
	HTTPClient           *http.Client
	TrustedDownloadHosts []string
	AllowHTTP            bool
}

type InstallerAcquireOptions struct {
	CacheDir             string
	HTTPClient           *http.Client
	SignatureVerifier    InstallerSignatureVerifier
	Acknowledgements     InstallerAcknowledgements
	TrustedDownloadHosts []string
	AllowHTTP            bool
}

type installerCacheMetadata struct {
	Version  int               `json:"version"`
	Artifact InstallerArtifact `json:"artifact"`
}

func ResolveLatestInstaller(
	ctx context.Context,
	variant InstallerVariant,
	options InstallerResolveOptions,
) (release InstallerRelease, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("resolve latest %s release: %w", installerDescription(variant), err)
		}
	}()
	if err := validateInstallerVariant(variant); err != nil {
		return InstallerRelease{}, err
	}
	if options.TagsURL == "" {
		options.TagsURL = DefaultTagsURL
	}
	if options.DownloadBaseURL == "" {
		options.DownloadBaseURL = DefaultDownloadBaseURL
	}
	if options.HTTPClient == nil {
		options.HTTPClient = http.DefaultClient
	}

	version, err := latestVersion(ctx, options.HTTPClient, options.TagsURL)
	if err != nil {
		return InstallerRelease{}, err
	}
	downloadURL, err := installerDownloadURL(options.DownloadBaseURL, version, variant)
	if err != nil {
		return InstallerRelease{}, err
	}
	if err := validateInstallerSource(downloadURL, options.TrustedDownloadHosts, options.AllowHTTP); err != nil {
		return InstallerRelease{}, err
	}
	return InstallerRelease{
		Version:   version,
		Variant:   variant,
		AssetName: installerFileName(version, variant),
		URL:       downloadURL,
	}, nil
}

func AcquireInstaller(
	ctx context.Context,
	release InstallerRelease,
	options InstallerAcquireOptions,
) (artifact InstallerArtifact, err error) {
	installerCacheMu.Lock()
	defer installerCacheMu.Unlock()

	defer func() {
		if err != nil {
			err = fmt.Errorf("acquire %s version %q: %w",
				installerDescription(release.Variant), release.Version, err)
		}
	}()
	if err := validateInstallerRelease(release, options.TrustedDownloadHosts, options.AllowHTTP); err != nil {
		return InstallerArtifact{}, err
	}
	if release.Variant == InstallerVariantAddon &&
		(!options.Acknowledgements.SinglePlayerAcknowledged ||
			!options.Acknowledgements.AntiCheatRiskAcknowledged) {
		return InstallerArtifact{}, errors.New(
			"full add-on installer requires separate single-player and anti-cheat risk acknowledgements")
	}
	if options.CacheDir == "" {
		options.CacheDir = DefaultInstallerCacheDir()
	}
	if options.HTTPClient == nil {
		options.HTTPClient = http.DefaultClient
	}
	if options.SignatureVerifier == nil {
		options.SignatureVerifier = platformInstallerSignatureVerifier{}
	}
	if err := os.MkdirAll(options.CacheDir, 0o755); err != nil {
		return InstallerArtifact{}, fmt.Errorf("create ReShade installer cache: %w", err)
	}

	installerPath := filepath.Join(options.CacheDir, release.AssetName)
	metadataPath := installerPath + ".json"
	if cached, cacheErr := readCachedInstaller(
		installerPath, metadataPath, release, options.SignatureVerifier,
	); cacheErr == nil {
		return cached, nil
	} else if !errors.Is(cacheErr, os.ErrNotExist) {
		_ = os.Remove(installerPath)
		_ = os.Remove(metadataPath)
	}

	if err := downloadInstaller(
		ctx,
		options.HTTPClient,
		release.URL,
		installerPath,
		options.TrustedDownloadHosts,
		options.AllowHTTP,
	); err != nil {
		return InstallerArtifact{}, err
	}
	artifact, err = inspectInstallerArtifact(installerPath, release, options.SignatureVerifier)
	if err != nil {
		_ = os.Remove(installerPath)
		return InstallerArtifact{}, err
	}
	if err := writeInstallerCacheMetadata(metadataPath, artifact); err != nil {
		_ = os.Remove(installerPath)
		return InstallerArtifact{}, err
	}
	return artifact, nil
}

func validateInstallerRelease(
	release InstallerRelease,
	trustedHosts []string,
	allowHTTP bool,
) error {
	if err := validateInstallerVariant(release.Variant); err != nil {
		return err
	}
	versionTag := "v" + strings.TrimSpace(release.Version)
	if !isCanonicalStableVersion(versionTag) {
		return fmt.Errorf("installer version %q is not a canonical stable version", release.Version)
	}
	expectedAssetName := installerFileName(release.Version, release.Variant)
	if release.AssetName != expectedAssetName {
		return fmt.Errorf("installer asset name %q does not match %q", release.AssetName, expectedAssetName)
	}
	if err := validateInstallerSource(release.URL, trustedHosts, allowHTTP); err != nil {
		return err
	}
	parsed, err := url.Parse(release.URL)
	if err != nil {
		return fmt.Errorf("parse installer URL: %w", err)
	}
	if filepath.Base(parsed.Path) != release.AssetName {
		return fmt.Errorf("installer URL path does not end with asset %q", release.AssetName)
	}
	return nil
}

func validateInstallerVariant(variant InstallerVariant) error {
	if variant != InstallerVariantStandard && variant != InstallerVariantAddon {
		return fmt.Errorf("installer variant %q is invalid", variant)
	}
	return nil
}

func validateInstallerSource(rawURL string, trustedHosts []string, allowHTTP bool) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse installer URL: %w", err)
	}
	if parsed.Scheme != "https" && !(allowHTTP && parsed.Scheme == "http") {
		return fmt.Errorf("installer URL %q must use HTTPS", rawURL)
	}
	if parsed.User != nil || parsed.Hostname() == "" {
		return fmt.Errorf("installer URL %q is not an allowed absolute URL", rawURL)
	}
	if len(trustedHosts) == 0 {
		trustedHosts = []string{"reshade.me"}
	}
	trusted := slices.ContainsFunc(trustedHosts, func(host string) bool {
		return strings.EqualFold(strings.TrimSpace(host), parsed.Hostname())
	})
	if !trusted {
		return fmt.Errorf("installer URL host %q is not trusted", parsed.Hostname())
	}
	return nil
}

func readCachedInstaller(
	installerPath string,
	metadataPath string,
	release InstallerRelease,
	verifier InstallerSignatureVerifier,
) (InstallerArtifact, error) {
	contents, err := os.ReadFile(metadataPath)
	if err != nil {
		return InstallerArtifact{}, err
	}
	var metadata installerCacheMetadata
	if err := json.Unmarshal(contents, &metadata); err != nil {
		return InstallerArtifact{}, fmt.Errorf("decode ReShade installer cache metadata: %w", err)
	}
	if metadata.Version != installerCacheMetadataVersion {
		return InstallerArtifact{}, fmt.Errorf(
			"ReShade installer cache metadata version %d is unsupported", metadata.Version)
	}
	if metadata.Artifact.InstallerRelease != release {
		return InstallerArtifact{}, errors.New("cached ReShade installer release metadata does not match")
	}
	inspected, err := inspectInstallerArtifact(installerPath, release, verifier)
	if err != nil {
		return InstallerArtifact{}, err
	}
	if inspected.SizeBytes != metadata.Artifact.SizeBytes ||
		!strings.EqualFold(inspected.SHA256, metadata.Artifact.SHA256) ||
		inspected.Signature != metadata.Artifact.Signature {
		return InstallerArtifact{}, errors.New("cached ReShade installer integrity metadata does not match")
	}
	return inspected, nil
}

func inspectInstallerArtifact(
	path string,
	release InstallerRelease,
	verifier InstallerSignatureVerifier,
) (InstallerArtifact, error) {
	hash, size, err := fileops.FileIntegrity(path)
	if err != nil {
		return InstallerArtifact{}, err
	}
	if size == 0 {
		return InstallerArtifact{}, errors.New("ReShade installer is empty")
	}
	signature, err := verifier.VerifyInstallerSignature(path, release.Variant)
	if err != nil {
		return InstallerArtifact{}, err
	}
	return InstallerArtifact{
		InstallerRelease: release,
		Path:             path,
		SizeBytes:        size,
		SHA256:           hash,
		Signature:        signature,
	}, nil
}

func downloadInstaller(
	ctx context.Context,
	client *http.Client,
	rawURL string,
	path string,
	trustedHosts []string,
	allowHTTP bool,
) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("build ReShade installer request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download ReShade installer: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("download ReShade installer: unexpected status %s", response.Status)
	}
	if response.Request == nil || response.Request.URL == nil {
		return errors.New("download ReShade installer: final response URL is missing")
	}
	if err := validateInstallerSource(response.Request.URL.String(), trustedHosts, allowHTTP); err != nil {
		return fmt.Errorf("validate final ReShade installer download URL: %w", err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary ReShade installer: %w", err)
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
		return fmt.Errorf("write ReShade installer: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close ReShade installer: %w", closeErr)
	}
	if written == 0 {
		return errors.New("download ReShade installer: empty response body")
	}
	if err := os.Rename(tempPath, path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("replace cached ReShade installer: %w", removeErr)
		}
		if err := os.Rename(tempPath, path); err != nil {
			return fmt.Errorf("cache ReShade installer: %w", err)
		}
	}
	removeTemp = false
	return nil
}

func writeInstallerCacheMetadata(path string, artifact InstallerArtifact) error {
	contents, err := json.MarshalIndent(installerCacheMetadata{
		Version:  installerCacheMetadataVersion,
		Artifact: artifact,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode ReShade installer cache metadata: %w", err)
	}
	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary ReShade installer cache metadata: %w", err)
	}
	tempPath := tempFile.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()
	if _, err := tempFile.Write(contents); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write ReShade installer cache metadata: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close ReShade installer cache metadata: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("replace ReShade installer cache metadata: %w", removeErr)
		}
		if err := os.Rename(tempPath, path); err != nil {
			return fmt.Errorf("commit ReShade installer cache metadata: %w", err)
		}
	}
	removeTemp = false
	return nil
}

func isCanonicalStableVersion(version string) bool {
	return strings.TrimSpace(version) == version &&
		strings.HasPrefix(version, "v") &&
		semverIsCanonicalStable(version)
}
