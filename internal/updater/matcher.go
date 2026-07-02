package updater

import (
	"strings"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func assetMatcher(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	filteredIndices := make([]int, 0, len(assets))
	filtered := make([]github.ReleaseAsset, 0, len(assets))
	for index, asset := range assets {
		name := strings.ToLower(asset.Name)
		if strings.HasSuffix(name, ".deb") ||
			strings.HasSuffix(name, ".rpm") ||
			strings.HasSuffix(name, ".appimage") ||
			strings.Contains(name, "installer") {
			continue
		}
		filteredIndices = append(filteredIndices, index)
		filtered = append(filtered, asset)
	}

	match := github.DefaultAssetMatcher(req, filtered)
	if match < 0 || match >= len(filteredIndices) {
		return -1
	}

	return filteredIndices[match]
}
