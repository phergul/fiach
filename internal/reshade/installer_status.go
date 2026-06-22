package reshade

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func ResolveInstallerStatus(ctx context.Context) InstallerStatus {
	return InstallerStatus{
		Standard: resolveInstallerVariantStatus(ctx, InstallerVariantStandard),
		Addon:    resolveInstallerVariantStatus(ctx, InstallerVariantAddon),
	}
}

func resolveInstallerVariantStatus(ctx context.Context, variant InstallerVariant) InstallerReleaseStatus {
	release, err := ResolveLatestInstaller(ctx, variant, InstallerResolveOptions{})
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
	}
	if digest, size, ok := cachedInstallerArtifactMetadata(release); ok {
		status.Digest = &digest
		status.Size = &size
		status.Cached = true
	}
	return status
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
