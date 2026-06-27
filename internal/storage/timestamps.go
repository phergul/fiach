package storage

import (
	"strings"
	"time"
)

const AppliedTimestampLayout = time.RFC3339

func FormatAppliedTimestamp(value time.Time) string {
	return value.UTC().Format(AppliedTimestampLayout)
}

func ParseAppliedTimestamp(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}

	parsed, err := time.Parse(AppliedTimestampLayout, value)
	if err != nil {
		return time.Time{}, false
	}

	return parsed, true
}
