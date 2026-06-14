package reshade

import (
	"fmt"

	"github.com/phergul/fiach/internal/filetxn"
)

func ResolveWithinRoot(root string, relativePath string) (resolved string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("resolve ReShade path inside game root: %w", err)
		}
	}()
	return filetxn.ResolveWithinRoot(root, relativePath)
}
