package iniedit

import (
	"strings"
	"testing"
)

func TestParsePreservingPreservesUnrelatedContentAndNewlines(t *testing.T) {
	t.Parallel()

	process := "Game.exe"
	input := []byte("; comment\r\n[Plugins]\r\nOther=true\r\nLoadReshade=false\r\n\r\n[Custom]\r\nValue=42\r\n")
	document, err := ParsePreserving(input)
	if err != nil {
		t.Fatalf("ParsePreserving() error = %v", err)
	}
	document.SetSingleKey("Plugins", "LoadReshade", "true")
	document.SetSingleKey("Spoofing", "Dxgi", "true")
	document.SetSingleKey("ProcessFilter", "TargetProcessName", process)
	document.SetSingleKey("Hotfix", "CheckForUpdate", "false")
	output := document.Bytes()
	text := string(output)
	for _, expected := range []string{
		"; comment\r\n", "Other=true", "[Custom]\r\nValue=42",
		"LoadReshade=true", "Dxgi=true", "TargetProcessName=Game.exe", "CheckForUpdate=false",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("Bytes() = %q, missing %q", text, expected)
		}
	}
	if strings.Contains(strings.ReplaceAll(text, "\r\n", ""), "\n") {
		t.Fatalf("Bytes() mixed newline styles: %q", text)
	}
}

func TestSetSingleKeyRemovesDuplicateManagedKeys(t *testing.T) {
	t.Parallel()

	document, err := ParsePreserving([]byte("[Plugins]\nLoadReshade=false\nloadreshade=false\n"))
	if err != nil {
		t.Fatalf("ParsePreserving() error = %v", err)
	}
	document.SetSingleKey("Plugins", "LoadReshade", "false")
	output := document.Bytes()
	if strings.Count(strings.ToLower(string(output)), "loadreshade=") != 1 {
		t.Fatalf("Bytes() = %q, want one managed key", output)
	}
}

func TestParsePreservingRejectsUTF16(t *testing.T) {
	t.Parallel()

	if _, err := ParsePreserving([]byte{0xff, 0xfe, 'x', 0}); err == nil {
		t.Fatal("ParsePreserving() error = nil, want unsupported encoding error")
	}
}

func newLFDocument(lines ...string) Document {
	return Document{
		lines:           append([]string(nil), lines...),
		newline:         "\n",
		hadFinalNewline: true,
	}
}

func TestSetCommaListKeyInsertsAndUpdates(t *testing.T) {
	t.Parallel()

	document := newLFDocument("[GENERAL]", "PresetPath=ReShadePreset.ini")
	document.SetCommaListKey("GENERAL", "EffectSearchPaths", []string{"reshade-shaders/Shaders", "mods/effects"})
	output := string(document.Bytes())
	if !strings.Contains(output, "EffectSearchPaths=reshade-shaders/Shaders,mods/effects") {
		t.Fatalf("Bytes() = %q, missing effect search paths", output)
	}

	document.SetCommaListKey("GENERAL", "EffectSearchPaths", []string{"mods/effects"})
	output = string(document.Bytes())
	if !strings.Contains(output, "EffectSearchPaths=mods/effects") {
		t.Fatalf("Bytes() = %q, want updated effect search paths", output)
	}
}

func TestSetCommaListKeyCreatesSection(t *testing.T) {
	t.Parallel()

	document := newLFDocument()
	document.SetCommaListKey("ADDON", "AddonPath", []string{"Addons"})
	output := string(document.Bytes())
	if !strings.Contains(output, "[ADDON]") || !strings.Contains(output, "AddonPath=Addons") {
		t.Fatalf("Bytes() = %q, want addon section and key", output)
	}
}
