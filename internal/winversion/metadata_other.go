//go:build !windows

package winversion

import "fmt"

func Read(string) (Metadata, error) {
	return Metadata{}, fmt.Errorf("read Windows version-resource metadata: unsupported platform")
}
