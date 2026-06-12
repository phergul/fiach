//go:build !windows

package optiscaler

import "fmt"

func InspectOwnership(path string) (ownership Ownership, err error) {
	return OwnershipUnknown, fmt.Errorf("inspect Windows version-resource ownership: unsupported platform")
}
