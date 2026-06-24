package reshade

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/iniedit"
)

func stageUpdatedReShadeINI(dataDir string, configPath string, effectPaths []string, texturePaths []string, addonPath string) (string, string, int64, error) {
	contents, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		contents = []byte("[GENERAL]\n")
	} else if err != nil {
		return "", "", 0, err
	}
	document, err := iniedit.ParseLF(contents)
	if err != nil {
		return "", "", 0, errors.New("ReShade.ini is not UTF-8 or ASCII")
	}
	if len(effectPaths) > 0 {
		document.SetCommaListKey("GENERAL", "EffectSearchPaths", deduplicateStrings(effectPaths))
	}
	if len(texturePaths) > 0 {
		document.SetCommaListKey("GENERAL", "TextureSearchPaths", deduplicateStrings(texturePaths))
	}
	if strings.TrimSpace(addonPath) != "" {
		document.SetCommaListKey("ADDON", "AddonPath", deduplicateStrings([]string{addonPath}))
	}
	updated := string(document.Bytes())
	stagePath := filepath.Join(dataDir, "staging", "generated", fileops.HashParts(configPath, updated)+".ini")
	if err := os.MkdirAll(filepath.Dir(stagePath), 0o755); err != nil {
		return "", "", 0, err
	}
	if err := os.WriteFile(stagePath, []byte(updated), 0o644); err != nil {
		return "", "", 0, err
	}
	hash, size, err := fileops.FileIntegrity(stagePath)
	return stagePath, hash, size, err
}

func deduplicateStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(filepath.Clean(value))
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}

func mergeSearchPaths(existing []contentPath, role PathRole, additions []string) []string {
	var values []string
	for _, item := range existing {
		if item.Role == role {
			values = append(values, item.Value)
		}
	}
	values = append(values, additions...)
	return deduplicateStrings(values)
}

func relativeINIPath(path string) string {
	return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(strings.ReplaceAll(path, "\\", string(filepath.Separator)))), "./")
}
