package storage

import (
	"testing"
	"time"
)

func TestFormatAndParseAppliedTimestamp(t *testing.T) {
	t.Parallel()

	value := time.Date(2026, 6, 27, 12, 30, 0, 0, time.UTC)
	formatted := FormatAppliedTimestamp(value)

	parsed, ok := ParseAppliedTimestamp(formatted)
	if !ok {
		t.Fatalf("ParseAppliedTimestamp(%q) ok = false, want true", formatted)
	}
	if !parsed.Equal(value) {
		t.Fatalf("ParseAppliedTimestamp() = %v, want %v", parsed, value)
	}
}

func TestParseAppliedTimestampRejectsEmpty(t *testing.T) {
	t.Parallel()

	if _, ok := ParseAppliedTimestamp(""); ok {
		t.Fatal("ParseAppliedTimestamp(\"\") ok = true, want false")
	}
}
