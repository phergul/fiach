package iniedit

import (
	"bytes"
	"errors"
	"strings"

	"github.com/phergul/fiach/internal/fileops"
)

var ErrInvalidText = errors.New("INI must use UTF-8 or ASCII text")

type Document struct {
	lines           []string
	newline         string
	hadFinalNewline bool
}

func ValidateText(contents []byte) error {
	if bytes.HasPrefix(contents, []byte{0xff, 0xfe}) || bytes.HasPrefix(contents, []byte{0xfe, 0xff}) {
		return ErrInvalidText
	}
	return validateUTF8(contents)
}

func ValidateUTF8(contents []byte) error {
	return validateUTF8(contents)
}

func validateUTF8(contents []byte) error {
	if !fileops.IsUTF8Text(contents) {
		return ErrInvalidText
	}
	return nil
}

func ParsePreserving(contents []byte) (Document, error) {
	if err := ValidateText(contents); err != nil {
		return Document{}, err
	}
	newline := "\n"
	if bytes.Contains(contents, []byte("\r\n")) {
		newline = "\r\n"
	}
	text := strings.ReplaceAll(string(contents), "\r\n", "\n")
	hadFinalNewline := strings.HasSuffix(text, "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if text == "" {
		lines = []string{}
	}
	return Document{
		lines:           lines,
		newline:         newline,
		hadFinalNewline: hadFinalNewline,
	}, nil
}

func ParseLF(contents []byte) (Document, error) {
	if err := ValidateUTF8(contents); err != nil {
		return Document{}, err
	}
	text := strings.ReplaceAll(string(contents), "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	lines := []string{}
	if text != "" {
		lines = strings.Split(text, "\n")
	}
	return Document{
		lines:           lines,
		newline:         "\n",
		hadFinalNewline: true,
	}, nil
}

func (d *Document) SetSingleKey(section string, key string, value string) {
	d.lines = setSingleKey(d.lines, section, key, value)
}

func (d *Document) SetCommaListKey(section string, key string, values []string) {
	d.lines = setCommaListKey(d.lines, section, key, values)
}

func (d Document) Bytes() []byte {
	result := strings.Join(d.lines, d.newline)
	if d.hadFinalNewline || len(d.lines) == 0 {
		result += d.newline
	}
	return []byte(result)
}

func (d Document) Lines() []string {
	return append([]string(nil), d.lines...)
}

func findSectionRange(lines []string, section string) (start int, end int) {
	start = -1
	end = len(lines)
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "[") || !strings.Contains(trimmed, "]") {
			continue
		}
		name := strings.TrimSpace(trimmed[1:strings.Index(trimmed, "]")])
		if strings.EqualFold(name, section) {
			if start < 0 {
				start = index
			}
			continue
		}
		if start >= 0 {
			end = index
			return start, end
		}
	}
	return start, end
}

func setSingleKey(lines []string, section string, key string, value string) []string {
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
			if strings.EqualFold(currentSection, section) {
				sectionStart = index
			}
			continue
		}
		if sectionStart >= 0 && strings.EqualFold(currentSection, section) {
			existingKey, _, ok := strings.Cut(trimmed, "=")
			if ok && strings.EqualFold(strings.TrimSpace(existingKey), key) {
				if !foundKey {
					prefix := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					lines[index] = prefix + key + "=" + value
					foundKey = true
				} else {
					lines = append(lines[:index], lines[index+1:]...)
					return setSingleKey(lines, section, key, value)
				}
			}
		}
	}
	if foundKey {
		return lines
	}
	entry := key + "=" + value
	if sectionStart >= 0 {
		return append(lines[:sectionEnd], append([]string{entry}, lines[sectionEnd:]...)...)
	}
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
		lines = append(lines, "")
	}
	return append(lines, "["+section+"]", entry)
}

func setCommaListKey(lines []string, section string, key string, values []string) []string {
	sectionStart, sectionEnd := findSectionRange(lines, section)
	entry := key + "=" + strings.Join(values, ",")
	if sectionStart < 0 {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		return append(lines, "["+section+"]", entry)
	}
	for index := sectionStart + 1; index < sectionEnd; index++ {
		line := strings.TrimSpace(lines[index])
		existingKey, _, ok := strings.Cut(line, "=")
		if ok && strings.EqualFold(strings.TrimSpace(existingKey), key) {
			lines[index] = entry
			return lines
		}
	}
	result := append([]string{}, lines[:sectionEnd]...)
	result = append(result, entry)
	result = append(result, lines[sectionEnd:]...)
	return result
}
