package reshade

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

func InspectPreset(path string, catalogue ContentCatalogue) (result PresetInspectionResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("inspect ReShade preset: %w", err)
		}
	}()
	contents, err := os.ReadFile(path)
	if err != nil {
		return PresetInspectionResult{}, err
	}
	if !utf8.Valid(contents) || bytes.IndexByte(contents, 0) >= 0 {
		return PresetInspectionResult{}, errors.New("preset is not UTF-8 or ASCII")
	}
	references := parsePresetEffectReferences(contents)
	result.ReferencedEffects = references
	remaining := map[string]bool{}
	for _, reference := range references {
		remaining[strings.ToLower(reference)] = true
	}
	for _, pkg := range catalogue.Effects {
		var matches []string
		for _, effect := range pkg.EffectFiles {
			if remaining[strings.ToLower(filepath.Base(effect))] {
				matches = append(matches, effect)
			}
		}
		if len(matches) == 0 {
			continue
		}
		sort.Strings(matches)
		result.Recommendations = append(result.Recommendations, PresetRecommendation{
			PackageID:   pkg.ID,
			PackageName: pkg.Name,
			EffectFiles: matches,
		})
		for _, match := range matches {
			delete(remaining, strings.ToLower(filepath.Base(match)))
		}
	}
	for _, reference := range references {
		if remaining[strings.ToLower(reference)] {
			result.MissingEffects = append(result.MissingEffects, reference)
		}
	}
	sort.Slice(result.Recommendations, func(i int, j int) bool {
		return result.Recommendations[i].PackageID < result.Recommendations[j].PackageID
	})
	return result, nil
}

func parsePresetEffectReferences(contents []byte) []string {
	seen := map[string]bool{}
	var result []string
	scanner := bufio.NewScanner(bytes.NewReader(contents))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "Techniques") {
			continue
		}
		for _, technique := range strings.Split(value, ",") {
			_, file, ok := strings.Cut(technique, "@")
			if !ok {
				continue
			}
			file = filepath.Base(strings.TrimSpace(file))
			if file == "" || seen[strings.ToLower(file)] {
				continue
			}
			seen[strings.ToLower(file)] = true
			result = append(result, file)
		}
	}
	sort.Strings(result)
	return result
}
