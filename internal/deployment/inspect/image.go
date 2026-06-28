package inspect

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/phergul/fiach/internal/fileops"
)

func readImageMetadata(path string) (*ImageMetadata, error) {
	_, err := fileops.StatRegularFile("image file", path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open image file %q: %w", path, err)
	}
	defer file.Close()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("decode image config %q: %w", path, err)
	}

	sha256Hex, sizeBytes, err := fileops.FileIntegrity(path)
	if err != nil {
		return nil, fmt.Errorf("hash image file %q: %w", path, err)
	}

	return &ImageMetadata{
		Format:    format,
		Width:     config.Width,
		Height:    config.Height,
		SHA256:    sha256Hex,
		SizeBytes: sizeBytes,
	}, nil
}

func imageMetadataOrFallback(path string, snapshot resolvedStatePath) (*ImageMetadata, string, error) {
	metadata, err := readImageMetadata(path)
	if err != nil {
		return &ImageMetadata{
			SHA256:    snapshot.SHA256,
			SizeBytes: snapshot.SizeBytes,
		}, "Image metadata could not be read; showing hash and size only.", nil
	}

	return metadata, "", nil
}
