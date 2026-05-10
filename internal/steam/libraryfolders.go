package steam

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/andygrunwald/vdf"
)

func ParseLibraryFolders(paths *SteamPaths) (libraries []string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("parse Steam library folders: %w", err)
		}
	}()

	if paths == nil {
		return nil, fmt.Errorf("Steam paths are not configured")
	}
	if paths.LibraryVDF == "" {
		return nil, fmt.Errorf("libraryfolders.vdf path is empty")
	}

	file, err := os.Open(paths.LibraryVDF)
	if err != nil {
		return nil, fmt.Errorf("open libraryfolders.vdf: %w", err)
	}
	defer file.Close()

	parser := vdf.NewParser(file)
	data, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse libraryfolders.vdf: %w", err)
	}

	root := toStringMap(data)
	libraryFoldersRaw, ok := root["libraryfolders"]
	if !ok {
		return nil, fmt.Errorf("missing libraryfolders key")
	}

	libraryFolders := toStringMap(libraryFoldersRaw)
	if libraryFolders == nil {
		return nil, fmt.Errorf("libraryfolders section is not a map")
	}

	libraries = make([]string, 0, len(libraryFolders)+1)
	libraries = append(libraries, paths.Root)

	for _, key := range sortedNumericKeys(libraryFolders) {
		entry := toStringMap(libraryFolders[key])
		if entry == nil {
			continue
		}

		path, ok := entry["path"].(string)
		if !ok || path == "" {
			continue
		}

		libraries = append(libraries, path)
	}

	return uniqueCleanPaths(libraries), nil
}

func sortedNumericKeys(m map[string]any) []string {
	type numericKey struct {
		key   string
		value int
	}

	keys := make([]numericKey, 0, len(m))
	for key := range m {
		if !isNumeric(key) {
			continue
		}

		value, err := strconv.Atoi(key)
		if err != nil {
			continue
		}

		keys = append(keys, numericKey{key: key, value: value})
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].value < keys[j].value
	})

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key.key)
	}

	return result
}

func uniqueCleanPaths(paths []string) []string {
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

func toStringMap(v any) map[string]any {
	switch m := v.(type) {
	case map[string]any:
		return m
	case map[any]any:
		out := make(map[string]any)
		for key, val := range m {
			keyStr, ok := key.(string)
			if !ok {
				continue
			}
			out[keyStr] = val
		}
		return out
	default:
		return nil
	}
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
