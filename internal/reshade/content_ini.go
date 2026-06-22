package reshade

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/phergul/fiach/internal/fileops"
)

func stageUpdatedReShadeINI(dataDir string, configPath string, effectPaths []string, texturePaths []string, addonPath string) (string, string, int64, error) {
	contents, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		contents = []byte("[GENERAL]\n")
	} else if err != nil {
		return "", "", 0, err
	}
	if !utf8.Valid(contents) || bytes.IndexByte(contents, 0) >= 0 {
		return "", "", 0, errors.New("ReShade.ini is not UTF-8 or ASCII")
	}
	lines := splitINILines(string(contents))
	if len(effectPaths) > 0 {
		lines = setReShadeINIList(lines, "GENERAL", "EffectSearchPaths", effectPaths)
	}
	if len(texturePaths) > 0 {
		lines = setReShadeINIList(lines, "GENERAL", "TextureSearchPaths", texturePaths)
	}
	if strings.TrimSpace(addonPath) != "" {
		lines = setReShadeINIList(lines, "ADDON", "AddonPath", []string{addonPath})
	}
	updated := strings.Join(lines, "\n")
	if !strings.HasSuffix(updated, "\n") {
		updated += "\n"
	}
	stagePath := filepath.Join(dataDir, "staging", "generated", contentHash(configPath, updated)+".ini")
	if err := os.MkdirAll(filepath.Dir(stagePath), 0o755); err != nil {
		return "", "", 0, err
	}
	if err := os.WriteFile(stagePath, []byte(updated), 0o644); err != nil {
		return "", "", 0, err
	}
	hash, size, err := fileIntegrity(stagePath)
	return stagePath, hash, size, err
}

func splitINILines(contents string) []string {
	contents = strings.ReplaceAll(contents, "\r\n", "\n")
	contents = strings.TrimSuffix(contents, "\n")
	if contents == "" {
		return []string{}
	}
	return strings.Split(contents, "\n")
}

func setReShadeINIList(lines []string, section string, key string, values []string) []string {
	sectionLower := strings.ToLower(section)
	keyLower := strings.ToLower(key)
	sectionStart := -1
	sectionEnd := len(lines)
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.Contains(trimmed, "]") {
			name := strings.TrimSpace(trimmed[1:strings.Index(trimmed, "]")])
			if strings.EqualFold(name, sectionLower) {
				sectionStart = index
				continue
			}
			if sectionStart >= 0 {
				sectionEnd = index
				break
			}
		}
	}
	value := key + "=" + strings.Join(deduplicateStrings(values), ",")
	if sectionStart < 0 {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		return append(lines, "["+section+"]", value)
	}
	for index := sectionStart + 1; index < sectionEnd; index++ {
		line := strings.TrimSpace(lines[index])
		existingKey, _, ok := strings.Cut(line, "=")
		if ok && strings.EqualFold(strings.TrimSpace(existingKey), keyLower) {
			lines[index] = value
			return lines
		}
	}
	result := append([]string{}, lines[:sectionEnd]...)
	result = append(result, value)
	result = append(result, lines[sectionEnd:]...)
	return result
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

func fileIntegrity(path string) (string, int64, error) {
	hash, size, err := fileops.FileIntegrity(path)
	if err != nil {
		return "", 0, fmt.Errorf("inspect generated file integrity: %w", err)
	}
	return hash, size, nil
}
