package updater

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

func TestBuiltinWindowUsesFiachTitle(t *testing.T) {
	window, ok := builtinWindow("ash").(*updater.BuiltinWindow)
	if !ok {
		t.Fatalf("builtinWindow() type = %T, want *updater.BuiltinWindow", builtinWindow("ash"))
	}

	if window.Options.Title != updaterWindowTitle {
		t.Fatalf("builtin window title = %q, want %q", window.Options.Title, updaterWindowTitle)
	}

	if window.CSS == "" {
		t.Fatal("builtin window css is empty")
	}
}
