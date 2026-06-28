package inspect

import (
	"fmt"

	"github.com/phergul/fiach/internal/fileops"
)

func binarySideMetadata(kind StateKind, resolved resolvedStatePath) SideMetadata {
	return SideMetadata{
		StateKind:         kind,
		Label:             stateLabel(kind),
		Available:         resolved.Available,
		UnavailableReason: resolved.Reason,
		SHA256:            resolved.SHA256,
		SizeBytes:         resolved.SizeBytes,
	}
}

func ensureHash(path string, resolved resolvedStatePath) (resolvedStatePath, error) {
	if resolved.SHA256 != "" && resolved.SizeBytes > 0 {
		return resolved, nil
	}

	sha256Hex, sizeBytes, err := fileops.FileIntegrity(path)
	if err != nil {
		return resolvedStatePath{}, fmt.Errorf("hash file %q: %w", path, err)
	}

	resolved.SHA256 = sha256Hex
	resolved.SizeBytes = sizeBytes
	return resolved, nil
}
