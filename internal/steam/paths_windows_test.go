//go:build windows

package steam

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestWindowsCommonSteamCandidates(t *testing.T) {
	t.Parallel()

	got := windowsCommonSteamCandidates(`C:`, `C:\Program Files (x86)`, `C:\Program Files`)
	want := []string{
		filepath.Join(`C:\Program Files (x86)`, "Steam"),
		filepath.Join(`C:\Program Files`, "Steam"),
		filepath.Join(`C:\`, "Program Files (x86)", "Steam"),
		filepath.Join(`C:\`, "Program Files", "Steam"),
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("windowsCommonSteamCandidates() = %#v, want %#v", got, want)
	}
}
