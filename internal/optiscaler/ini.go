package optiscaler

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

type iniKey struct {
	Section string
	Key     string
	Value   string
}

func UpdateManagedINI(contents []byte, config ManagedConfig) (updated []byte, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update OptiScaler INI settings: %w", err)
		}
	}()

	if bytes.HasPrefix(contents, []byte{0xff, 0xfe}) || bytes.HasPrefix(contents, []byte{0xfe, 0xff}) ||
		!utf8.Valid(contents) || bytes.IndexByte(contents, 0) >= 0 {
		return nil, errors.New("INI must use UTF-8 or ASCII text")
	}
	newline := "\n"
	if bytes.Contains(contents, []byte("\r\n")) {
		newline = "\r\n"
	}
	text := strings.ReplaceAll(string(contents), "\r\n", "\n")
	hadFinalNewline := strings.HasSuffix(text, "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	keys := []iniKey{
		{
			Section: "Plugins",
			Key:     "LoadReshade",
			Value:   boolINI(config.LoadReShade),
		},
		{
			Section: "Spoofing",
			Key:     "Dxgi",
			Value:   boolINI(config.DXGISpoofing),
		},
		{
			Section: "ProcessFilter",
			Key:     "TargetProcessName",
			Value:   optionalINI(config.TargetProcessName),
		},
		{
			Section: "Hotfix",
			Key:     "CheckForUpdate",
			Value:   "false",
		},
	}
	for _, key := range keys {
		lines = setINIKey(lines, key)
	}
	result := strings.Join(lines, newline)
	if hadFinalNewline || len(contents) == 0 {
		result += newline
	}
	return []byte(result), nil
}

func setINIKey(lines []string, managed iniKey) []string {
	sectionStart, sectionEnd := -1, len(lines)
	currentSection := ""
	foundKey := false
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if sectionStart >= 0 {
				sectionEnd = index
				break
			}
			currentSection = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			if strings.EqualFold(currentSection, managed.Section) {
				sectionStart = index
			}
			continue
		}
		if sectionStart >= 0 && strings.EqualFold(currentSection, managed.Section) {
			key, _, ok := strings.Cut(trimmed, "=")
			if ok && strings.EqualFold(strings.TrimSpace(key), managed.Key) {
				if !foundKey {
					prefix := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					lines[index] = prefix + managed.Key + "=" + managed.Value
					foundKey = true
				} else {
					lines = append(lines[:index], lines[index+1:]...)
					return setINIKey(lines, managed)
				}
			}
		}
	}
	if foundKey {
		return lines
	}
	entry := managed.Key + "=" + managed.Value
	if sectionStart >= 0 {
		return append(lines[:sectionEnd], append([]string{entry}, lines[sectionEnd:]...)...)
	}
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
		lines = append(lines, "")
	}
	return append(lines, "["+managed.Section+"]", entry)
}

func boolINI(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func optionalINI(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
