package reshade

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/fiach/internal/fileops"
)

func DecodeManifest(value string) (manifest Manifest, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("decode ReShade ownership manifest: %w", err)
		}
	}()
	if err := json.Unmarshal([]byte(value), &manifest); err != nil {
		return Manifest{}, err
	}
	if manifest.Version != ManifestVersion {
		return Manifest{}, fmt.Errorf("unsupported manifest version %d", manifest.Version)
	}
	for _, file := range manifest.Files {
		if file.RelativePath == "" || filepath.IsAbs(file.RelativePath) {
			return Manifest{}, errors.New("manifest file path must be relative")
		}
		clean := filepath.Clean(file.RelativePath)
		if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return Manifest{}, errors.New("manifest file path escapes target")
		}
		if file.SizeBytes < 0 || strings.TrimSpace(file.SHA256) == "" {
			return Manifest{}, errors.New("manifest file integrity is invalid")
		}
		switch file.Ownership {
		case OwnershipManaged, OwnershipAdopted, OwnershipUser, OwnershipForeign:
		default:
			return Manifest{}, fmt.Errorf("manifest file ownership %q is invalid", file.Ownership)
		}
	}
	return manifest, nil
}

func detectManifestDrift(targetPath string, manifest Manifest) (drift []Drift, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("detect ReShade managed-file drift: %w", err)
		}
	}()
	for _, file := range manifest.Files {
		if file.Ownership != OwnershipManaged && file.Ownership != OwnershipAdopted {
			continue
		}
		path := filepath.Join(targetPath, file.RelativePath)
		hash, size, hashErr := fileops.FileIntegrity(path)
		if hashErr != nil {
			if !errors.Is(hashErr, os.ErrNotExist) {
				return nil, hashErr
			}
			drift = append(drift, Drift{
				RelativePath: file.RelativePath,
				ExpectedHash: file.SHA256,
				Missing:      true,
			})
			continue
		}
		if !strings.EqualFold(hash, file.SHA256) || size != file.SizeBytes {
			drift = append(drift, Drift{
				RelativePath: file.RelativePath,
				ExpectedHash: file.SHA256,
				ActualHash:   hash,
			})
		}
	}
	return drift, nil
}
