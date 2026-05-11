package steam

import (
	"fmt"
	"path/filepath"
	"strings"
)

type ImageType string

const (
	ImageTypeLogo   ImageType = "logo"
	ImageTypeBanner ImageType = "banner"
	ImageTypeHero   ImageType = "hero"
)

func ResolveGameImagePath(artworkRoot string, appID string, imageType ImageType) (imagePath string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("resolve Steam game image: %w", err)
		}
	}()

	artworkRoot = strings.TrimSpace(artworkRoot)
	if artworkRoot == "" {
		return "", fmt.Errorf("artwork root is empty")
	}

	appID = strings.TrimSpace(appID)
	if appID == "" {
		return "", fmt.Errorf("app ID is empty")
	}

	gameArtworkPath := filepath.Join(artworkRoot, appID)
	if !dirExists(gameArtworkPath) {
		return "", nil
	}

	switch imageType {
	case ImageTypeLogo:
		return firstExistingFile(
			filepath.Join(gameArtworkPath, "logo.jpg"),
			filepath.Join(gameArtworkPath, "logo.png"),
			filepath.Join(gameArtworkPath, "*", "logo.jpg"),
			filepath.Join(gameArtworkPath, "*", "logo.png"),
		), nil
	case ImageTypeBanner:
		return firstExistingFile(
			filepath.Join(gameArtworkPath, "library_600x900.jpg"),
			filepath.Join(gameArtworkPath, "library_600x900.png"),
			filepath.Join(gameArtworkPath, "*", "library_600x900.jpg"),
			filepath.Join(gameArtworkPath, "*", "library_600x900.png"),
			filepath.Join(gameArtworkPath, "library_capsule.jpg"),
			filepath.Join(gameArtworkPath, "library_capsule.png"),
			filepath.Join(gameArtworkPath, "*", "library_capsule.jpg"),
			filepath.Join(gameArtworkPath, "*", "library_capsule.png"),
		), nil
	case ImageTypeHero:
		return firstExistingFile(
			filepath.Join(gameArtworkPath, "library_hero.jpg"),
			filepath.Join(gameArtworkPath, "library_hero.png"),
			filepath.Join(gameArtworkPath, "*", "library_hero.jpg"),
			filepath.Join(gameArtworkPath, "*", "library_hero.png"),
		), nil
	default:
		return "", fmt.Errorf("unsupported image type %q", imageType)
	}
}

func firstExistingFile(patterns ...string) string {
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			if fileExists(match) {
				return match
			}
		}
	}

	return ""
}
