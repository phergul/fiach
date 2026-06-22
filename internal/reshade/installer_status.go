package reshade

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/phergul/fiach/internal/thirdparty"
)

func ResolveInstallerStatus(ctx context.Context, refresh bool) InstallerStatus {
	result, warning, err := loadInstallerStatusManifest(ctx, refresh)
	if err != nil {
		message := err.Error()
		return InstallerStatus{
			Standard: InstallerReleaseStatus{
				Variant: InstallerVariantStandard,
				Error:   message,
			},
			Addon: InstallerReleaseStatus{
				Variant: InstallerVariantAddon,
				Error:   message,
			},
		}
	}
	return InstallerStatus{
		Standard: resolveInstallerVariantStatus(ctx, InstallerVariantStandard, result, warning),
		Addon:    resolveInstallerVariantStatus(ctx, InstallerVariantAddon, result, warning),
	}
}

func resolveInstallerVariantStatus(
	ctx context.Context,
	variant InstallerVariant,
	manifest thirdparty.Manifest,
	warning string,
) InstallerReleaseStatus {
	_ = ctx
	entry := manifest.Tools.ReShade.Standard
	if variant == InstallerVariantAddon {
		entry = manifest.Tools.ReShade.Addon
	}
	release := InstallerRelease{
		Version:   entry.Version,
		Variant:   variant,
		AssetName: entry.AssetName,
		URL:       entry.URL,
		SHA256:    entry.SHA256,
		SizeBytes: entry.SizeBytes,
	}
	err := validateInstallerRelease(release, nil, false)
	if err != nil {
		return InstallerReleaseStatus{
			Variant: variant,
			Error:   err.Error(),
		}
	}
	status := InstallerReleaseStatus{
		Version:   release.Version,
		Variant:   release.Variant,
		AssetName: release.AssetName,
		URL:       release.URL,
		Digest:    &release.SHA256,
		Size:      &release.SizeBytes,
		Error:     warning,
	}
	if digest, size, ok := cachedInstallerArtifactMetadata(release); ok {
		status.Digest = &digest
		status.Size = &size
		status.Cached = true
	}
	return status
}

func loadInstallerStatusManifest(ctx context.Context, refresh bool) (thirdparty.Manifest, string, error) {
	result, err := thirdparty.Load(ctx, thirdparty.LoadOptions{
		Refresh:    refresh,
		HTTPClient: http.DefaultClient,
	})
	if err != nil {
		return thirdparty.Manifest{}, "", err
	}
	return result.Manifest, result.Warning, nil
}

func cachedInstallerArtifactMetadata(release InstallerRelease) (string, int64, bool) {
	metadataPath := filepath.Join(DefaultInstallerCacheDir(), release.AssetName+".json")
	contents, err := os.ReadFile(metadataPath)
	if err != nil {
		return "", 0, false
	}
	metadata, err := decodeInstallerCacheMetadata(contents)
	if err != nil || metadata.Artifact.InstallerRelease != release {
		return "", 0, false
	}
	installerPath := filepath.Join(DefaultInstallerCacheDir(), release.AssetName)
	info, err := os.Stat(installerPath)
	if err != nil || info.Size() <= 0 {
		return "", 0, false
	}
	return metadata.Artifact.SHA256, metadata.Artifact.SizeBytes, true
}

func decodeInstallerCacheMetadata(contents []byte) (installerCacheMetadata, error) {
	var metadata installerCacheMetadata
	if err := json.Unmarshal(contents, &metadata); err != nil {
		return installerCacheMetadata{}, err
	}
	if metadata.Version != installerCacheMetadataVersion {
		return installerCacheMetadata{}, errors.New("installer cache metadata version is unsupported")
	}
	return metadata, nil
}
