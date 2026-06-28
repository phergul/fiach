package inspect

import (
	"fmt"
	"os"
	"strings"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func buildTextDiff(leftPath string, rightPath string) ([]TextDiffLine, bool, string, error) {
	leftContents, leftLimited, leftReason, err := readTextFile(leftPath)
	if err != nil {
		return nil, false, "", err
	}
	rightContents, rightLimited, rightReason, err := readTextFile(rightPath)
	if err != nil {
		return nil, false, "", err
	}

	if leftLimited || rightLimited {
		reason := firstNonEmpty(leftReason, rightReason, "Text diff is unavailable because one or both files exceed the size limit.")
		return nil, true, reason, nil
	}

	lines := diffTextLines(leftContents, rightContents)
	return lines, false, "", nil
}

func diffTextLines(left string, right string) []TextDiffLine {
	dmp := diffmatchpatch.New()
	leftRunes, rightRunes, lineArray := dmp.DiffLinesToRunes(left, right)
	diffs := dmp.DiffMainRunes(leftRunes, rightRunes, false)
	diffs = dmp.DiffCleanupSemantic(diffs)

	lines := make([]TextDiffLine, 0)
	leftLineNo := 1
	rightLineNo := 1

	for _, diff := range diffs {
		text := restoreDiffText(diff.Text, lineArray)
		chunks := strings.Split(text, "\n")
		if len(chunks) > 0 && chunks[len(chunks)-1] == "" {
			chunks = chunks[:len(chunks)-1]
		}

		for _, chunk := range chunks {
			switch diff.Type {
			case diffmatchpatch.DiffEqual:
				lines = append(lines, TextDiffLine{
					Kind:   "equal",
					Line:   chunk,
					LineNo: leftLineNo,
				})
				leftLineNo++
				rightLineNo++
			case diffmatchpatch.DiffDelete:
				lines = append(lines, TextDiffLine{
					Kind:   "delete",
					Line:   chunk,
					LineNo: leftLineNo,
				})
				leftLineNo++
			case diffmatchpatch.DiffInsert:
				lines = append(lines, TextDiffLine{
					Kind:   "insert",
					Line:   chunk,
					LineNo: rightLineNo,
				})
				rightLineNo++
			}
		}
	}

	return lines
}

func restoreDiffText(encoded string, lineArray []string) string {
	if len(lineArray) == 0 {
		return encoded
	}

	var builder strings.Builder
	for _, char := range encoded {
		index := int(char)
		if index >= 0 && index < len(lineArray) {
			builder.WriteString(lineArray[index])
		}
	}

	return builder.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func readTextFile(path string) (string, bool, string, error) {
	info, err := fileops.StatRegularFile("text file", path)
	if err != nil {
		return "", false, "", err
	}
	if info.Size() > MaxTextDiffBytes {
		return "", true, fmt.Sprintf("File exceeds the %d byte text diff limit.", MaxTextDiffBytes), nil
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return "", false, "", fmt.Errorf("read text file %q: %w", path, err)
	}
	if !fileops.IsUTF8Text(contents) {
		return "", true, "File is not readable UTF-8 text.", nil
	}

	return string(contents), false, "", nil
}
