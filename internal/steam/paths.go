package steam

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrSteamNotFound = errors.New("steam installation not found")

var steamRootCandidates = defaultSteamRootCandidates

type SteamPaths struct {
	Root       string
	SteamApps  string
	UserData   string
	LibraryVDF string
	Artwork    string
}

func FindSteamPaths(manualPath string) (*SteamPaths, error) {
	manualPath = strings.TrimSpace(manualPath)
	if manualPath != "" {
		paths, err := ValidateSteamRoot(manualPath)
		if err != nil {
			return nil, fmt.Errorf("manual Steam path %q is invalid: %w", manualPath, err)
		}

		return paths, nil
	}

	for _, candidate := range uniqueNonEmpty(steamRootCandidates()) {
		paths, err := ValidateSteamRoot(candidate)
		if err == nil {
			return paths, nil
		}
	}

	return nil, fmt.Errorf("%w: checked common Steam install locations", ErrSteamNotFound)
}

func ValidateSteamRoot(root string) (*SteamPaths, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("%w: Steam root path is empty", ErrSteamNotFound)
	}

	root = filepath.Clean(root)
	steamApps := filepath.Join(root, "steamapps")
	libraryVDF := filepath.Join(steamApps, "libraryfolders.vdf")
	userData := filepath.Join(root, "userdata")

	if !dirExists(root) {
		return nil, fmt.Errorf("%w: Steam root %q does not exist", ErrSteamNotFound, root)
	}
	if !dirExists(steamApps) {
		return nil, fmt.Errorf("%w: Steam root %q is missing steamapps directory", ErrSteamNotFound, root)
	}
	if !fileExists(libraryVDF) {
		return nil, fmt.Errorf("%w: Steam root %q is missing steamapps/libraryfolders.vdf", ErrSteamNotFound, root)
	}
	if !dirExists(userData) {
		return nil, fmt.Errorf("%w: Steam root %q is missing userdata directory", ErrSteamNotFound, root)
	}

	return &SteamPaths{
		Root:       root,
		SteamApps:  steamApps,
		UserData:   userData,
		LibraryVDF: libraryVDF,
		Artwork:    filepath.Join(root, "appcache", "librarycache"),
	}, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func uniqueNonEmpty(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		cleanPath := filepath.Clean(path)
		if _, ok := seen[cleanPath]; ok {
			continue
		}

		seen[cleanPath] = struct{}{}
		result = append(result, cleanPath)
	}

	return result
}
