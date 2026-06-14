//go:build windows

package optiscaler

import (
	"fmt"
	"strings"

	"github.com/phergul/fiach/internal/winversion"
)

func InspectOwnership(path string) (ownership Ownership, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("inspect Windows version-resource ownership: %w", err)
		}
	}()

	metadata, err := winversion.Read(path)
	if err != nil {
		return OwnershipUnknown, err
	}
	values := []string{
		metadata.CompanyName,
		metadata.FileDescription,
		metadata.InternalName,
		metadata.OriginalFilename,
		metadata.ProductName,
	}
	joined := strings.ToLower(strings.Join(values, "\n"))
	hasOptiScaler := strings.Contains(joined, "optiscaler")
	hasReShade := strings.EqualFold(strings.TrimSpace(metadata.ProductName), "ReShade") &&
		strings.EqualFold(strings.TrimSpace(metadata.OriginalFilename), "ReShade64.dll")
	switch {
	case hasOptiScaler && !hasReShade:
		return OwnershipOptiScaler, nil
	case hasReShade && !hasOptiScaler:
		return OwnershipReShade, nil
	default:
		return OwnershipUnknown, nil
	}
}
