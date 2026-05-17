package installconfig

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
)

const TargetBaseGameRoot = "game_root"

var windowsDriveTargetPath = regexp.MustCompile(`^[A-Za-z]:`)

func NormalizeTargetRelativePath(targetRelativePath string) (string, error) {
	targetRelativePath = strings.TrimSpace(targetRelativePath)
	if targetRelativePath == "" {
		return "", errors.New("target relative path is required")
	}

	targetRelativePath = strings.ReplaceAll(targetRelativePath, "\\", "/")
	if strings.HasPrefix(targetRelativePath, "/") || path.IsAbs(targetRelativePath) || windowsDriveTargetPath.MatchString(targetRelativePath) {
		return "", fmt.Errorf("target path %q must be relative to the game folder", targetRelativePath)
	}

	cleanPath := path.Clean(targetRelativePath)
	if cleanPath == "/" || cleanPath == "" {
		return "", fmt.Errorf("target path %q is not valid", targetRelativePath)
	}
	if cleanPath == ".." || strings.HasPrefix(cleanPath, "../") || strings.Contains(cleanPath, "/../") {
		return "", fmt.Errorf("target path %q escapes the game folder", targetRelativePath)
	}

	return cleanPath, nil
}

func DisplayTargetRelativePath(targetRelativePath string) string {
	if targetRelativePath == "." {
		return "Game root"
	}

	return targetRelativePath
}
