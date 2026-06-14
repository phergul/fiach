package optiscaler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackageInventoryExcludesSetupScriptsAndExtractionReadme(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for name, contents := range map[string]string{
		"OptiScaler.dll":         "runtime",
		"OptiScaler.ini":         "config",
		"LICENSE.txt":            "license",
		"setup.bat":              "script",
		"Configure.cmd":          "script",
		"Setup Instructions.txt": "instructions",
		"Extraction README.txt":  "instructions",
		"README.md":              "instructions",
		"!! README_EXTRACT ALL FILES TO GAME FOLDER !!.txt": "instructions",
		"setup_linux.sh": "script",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", name, err)
		}
	}

	files, err := packageInventory(root)
	if err != nil {
		t.Fatalf("packageInventory() error = %v", err)
	}

	got := map[string]bool{}
	for _, file := range files {
		got[filepath.Base(file.RelativePath)] = true
	}
	for _, name := range []string{"OptiScaler.dll", "OptiScaler.ini", "LICENSE.txt"} {
		if !got[name] {
			t.Fatalf("package inventory missing required asset %q: %#v", name, got)
		}
	}
	for _, name := range []string{
		"setup.bat",
		"Configure.cmd",
		"Setup Instructions.txt",
		"Extraction README.txt",
		"README.md",
		"!! README_EXTRACT ALL FILES TO GAME FOLDER !!.txt",
		"setup_linux.sh",
	} {
		if got[name] {
			t.Fatalf("package inventory included excluded asset %q", name)
		}
	}
}
