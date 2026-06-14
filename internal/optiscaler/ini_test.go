package optiscaler

import (
	"strings"
	"testing"
)

func TestUpdateManagedINIPreservesUnrelatedContentAndNewlines(t *testing.T) {
	t.Parallel()

	process := "Game.exe"
	input := []byte("; comment\r\n[Plugins]\r\nOther=true\r\nLoadReshade=false\r\n\r\n[Custom]\r\nValue=42\r\n")
	output, err := UpdateManagedINI(input, ManagedConfig{
		LoadReShade:       true,
		DXGISpoofing:      true,
		TargetProcessName: &process,
	})
	if err != nil {
		t.Fatalf("UpdateManagedINI() error = %v", err)
	}
	text := string(output)
	for _, expected := range []string{
		"; comment\r\n", "Other=true", "[Custom]\r\nValue=42",
		"LoadReshade=true", "Dxgi=true", "TargetProcessName=Game.exe", "CheckForUpdate=false",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("UpdateManagedINI() = %q, missing %q", text, expected)
		}
	}
	if strings.Contains(strings.ReplaceAll(text, "\r\n", ""), "\n") {
		t.Fatalf("UpdateManagedINI() mixed newline styles: %q", text)
	}
}

func TestUpdateManagedINIRemovesDuplicateManagedKeys(t *testing.T) {
	t.Parallel()

	output, err := UpdateManagedINI([]byte("[Plugins]\nLoadReshade=false\nloadreshade=false\n"), ManagedConfig{})
	if err != nil {
		t.Fatalf("UpdateManagedINI() error = %v", err)
	}
	if strings.Count(strings.ToLower(string(output)), "loadreshade=") != 1 {
		t.Fatalf("UpdateManagedINI() = %q, want one managed key", output)
	}
}

func TestUpdateManagedINIRejectsUTF16(t *testing.T) {
	t.Parallel()

	if _, err := UpdateManagedINI([]byte{0xff, 0xfe, 'x', 0}, ManagedConfig{}); err == nil {
		t.Fatal("UpdateManagedINI() error = nil, want unsupported encoding error")
	}
}
