package reshade

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/phergul/fiach/internal/fileops"
)

type contentPath struct {
	Value string
	Role  PathRole
}

func inventoryUserContent(gameRoot string, targetPath string) ([]UserContent, []string, error) {
	var warnings []string
	content := []UserContent{}
	configPath := filepath.Join(targetPath, "ReShade.ini")
	config, exists, err := inventoryContentFile(gameRoot, configPath, PathRoleConfiguration)
	if err != nil {
		return nil, nil, err
	}
	if exists {
		content = append(content, config)
	}
	if !exists {
		content = deduplicateUserContent(content)
		return content, warnings, nil
	}
	paths, parseWarnings, err := parseReShadeContentPaths(configPath)
	if err != nil {
		return nil, nil, err
	}
	warnings = append(warnings, parseWarnings...)
	for _, item := range paths {
		resolved := item.Value
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(targetPath, resolved)
		}
		resolved = filepath.Clean(resolved)
		if fileops.RequirePathWithinRoot("ReShade user content", resolved, gameRoot) != nil {
			content = append(content, UserContent{
				Path:          resolved,
				Role:          item.Role,
				Exists:        pathExists(resolved),
				External:      true,
				Directory:     item.Role != PathRolePreset,
				InventoryOnly: true,
			})
			warnings = append(warnings,
				fmt.Sprintf("External %s path %q is preserved but not traversed.", item.Role, resolved))
			continue
		}
		info, statErr := os.Stat(resolved)
		if errors.Is(statErr, os.ErrNotExist) {
			continue
		}
		if statErr != nil {
			return nil, nil, statErr
		}
		if info.Mode().IsRegular() {
			entry, _, inventoryErr := inventoryContentFile(gameRoot, resolved, item.Role)
			if inventoryErr != nil {
				return nil, nil, inventoryErr
			}
			content = append(content, entry)
			continue
		}
		if !info.IsDir() {
			warnings = append(warnings,
				fmt.Sprintf("ReShade %s path %q is neither a regular file nor directory.", item.Role, resolved))
			continue
		}
		if item.Role == PathRoleScreenshots {
			relative, relativeErr := filepath.Rel(gameRoot, resolved)
			if relativeErr != nil {
				return nil, nil, relativeErr
			}
			content = append(content, UserContent{
				Path:          relative,
				Role:          item.Role,
				Exists:        true,
				Directory:     true,
				InventoryOnly: true,
			})
			continue
		}
		walkErr := filepath.WalkDir(resolved, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			inventoried, _, inventoryErr := inventoryContentFile(gameRoot, path, item.Role)
			if inventoryErr != nil {
				return inventoryErr
			}
			content = append(content, inventoried)
			return nil
		})
		if walkErr != nil {
			return nil, nil, walkErr
		}
	}
	content = deduplicateUserContent(content)
	sort.Slice(content, func(i int, j int) bool {
		if content[i].Role != content[j].Role {
			return content[i].Role < content[j].Role
		}
		return strings.ToLower(content[i].Path) < strings.ToLower(content[j].Path)
	})
	return content, warnings, nil
}

func parseReShadeContentPaths(configPath string) ([]contentPath, []string, error) {
	contents, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		return defaultReShadeContentPaths(), nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if !utf8.Valid(contents) || bytes.IndexByte(contents, 0) >= 0 {
		return defaultReShadeContentPaths(),
			[]string{"ReShade.ini is not UTF-8 or ASCII; only default content paths were inventoried."},
			nil
	}
	values := map[string][]string{}
	section := ""
	scanner := bufio.NewScanner(bytes.NewReader(contents))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[section+"."+strings.ToLower(strings.TrimSpace(key))] =
			splitReShadePaths(strings.TrimSpace(value))
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	var result []contentPath
	appendValues := func(key string, role PathRole) {
		for _, value := range values[key] {
			result = append(result, contentPath{
				Value: value,
				Role:  role,
			})
		}
	}
	appendValues("general.presetpath", PathRolePreset)
	appendValues("general.effectsearchpaths", PathRoleEffects)
	appendValues("general.texturesearchpaths", PathRoleTextures)
	appendValues("addon.addonpath", PathRoleAddons)
	appendValues("screenshot.savepath", PathRoleScreenshots)
	if len(result) == 0 {
		return defaultReShadeContentPaths(), nil, nil
	}
	return result, nil, nil
}

func defaultReShadeContentPaths() []contentPath {
	return []contentPath{
		{
			Value: "ReShadePreset.ini",
			Role:  PathRolePreset,
		},
		{
			Value: filepath.Join("reshade-shaders", "Shaders"),
			Role:  PathRoleEffects,
		},
		{
			Value: filepath.Join("reshade-shaders", "Textures"),
			Role:  PathRoleTextures,
		},
		{
			Value: "Addons",
			Role:  PathRoleAddons,
		},
		{
			Value: ".",
			Role:  PathRoleScreenshots,
		},
	}
}

func splitReShadePaths(value string) []string {
	var result []string
	for _, path := range strings.Split(value, ",") {
		path = strings.ReplaceAll(strings.Trim(strings.TrimSpace(path), `"`), "\\", string(filepath.Separator))
		if path != "" {
			result = append(result, path)
		}
	}
	return result
}

func inventoryContentFile(gameRoot string, path string, role PathRole) (UserContent, bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return UserContent{}, false, nil
	}
	if err != nil {
		return UserContent{}, false, err
	}
	if !info.Mode().IsRegular() {
		return UserContent{}, false, nil
	}
	relative, err := filepath.Rel(gameRoot, path)
	if err != nil {
		return UserContent{}, false, err
	}
	hash, size, err := fileops.FileIntegrity(path)
	if err != nil {
		return UserContent{}, false, err
	}
	return UserContent{
		Path:      relative,
		Role:      role,
		SHA256:    hash,
		SizeBytes: size,
		Exists:    true,
	}, true, nil
}

func deduplicateUserContent(content []UserContent) []UserContent {
	result := make([]UserContent, 0, len(content))
	seen := map[string]bool{}
	for _, item := range content {
		key := string(item.Role) + ":" + strings.ToLower(filepath.Clean(item.Path))
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, item)
	}
	return result
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
