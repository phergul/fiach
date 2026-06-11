package unrealpak

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectAcceptsPakAndIoStoreGroupsAndFlattensFiles(t *testing.T) {
	t.Parallel()

	root := makeSource(t, map[string]string{
		"nested/Alpha_P.pak": "pak",
		"other/Beta_P.ucas":  "ucas",
		"other/Beta_P.utoc":  "utoc",
		"other/readme.txt":   "ignored",
	})

	inspection, err := Inspect(root)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	gotNames := make([]string, 0, len(inspection.Files))
	for _, file := range inspection.Files {
		gotNames = append(gotNames, file.Name)
	}
	wantNames := []string{"Alpha_P.pak", "Beta_P.ucas", "Beta_P.utoc"}
	if strings.Join(gotNames, "|") != strings.Join(wantNames, "|") {
		t.Fatalf("Inspect() names = %v, want %v", gotNames, wantNames)
	}
	if inspection.SizeBytes != int64(len("pak")+len("ucas")+len("utoc")) {
		t.Fatalf("Inspect() size = %d, want recognized file size", inspection.SizeBytes)
	}
	if len(inspection.Warnings) != 1 || !strings.Contains(inspection.Warnings[0], "Ignored 1") {
		t.Fatalf("Inspect() warnings = %v, want ignored-file warning", inspection.Warnings)
	}
}

func TestInspectRejectsIncompleteIoStoreGroup(t *testing.T) {
	t.Parallel()

	root := makeSource(t, map[string]string{
		"Broken_P.pak":  "pak",
		"Broken_P.ucas": "ucas",
	})

	_, err := Inspect(root)
	if err == nil || !strings.Contains(err.Error(), ".utoc") {
		t.Fatalf("Inspect() error = %v, want missing .utoc", err)
	}
}

func TestInspectRejectsFlattenedNameCollision(t *testing.T) {
	t.Parallel()

	root := makeSource(t, map[string]string{
		"first/Mod_P.pak":  "one",
		"second/mod_p.PAK": "two",
	})

	_, err := Inspect(root)
	if err == nil || !strings.Contains(err.Error(), "both flatten") {
		t.Fatalf("Inspect() error = %v, want flatten collision", err)
	}
}

func TestInspectRejectsInconsistentGroupStemCasing(t *testing.T) {
	t.Parallel()

	root := makeSource(t, map[string]string{
		"Mod_P.ucas": "ucas",
		"mod_p.utoc": "utoc",
	})

	_, err := Inspect(root)
	if err == nil || !strings.Contains(err.Error(), "inconsistent stem casing") {
		t.Fatalf("Inspect() error = %v, want stem casing rejection", err)
	}
}

func TestInspectWarnsWhenGroupLacksPatchSuffix(t *testing.T) {
	t.Parallel()

	root := makeSource(t, map[string]string{"Mod.pak": "pak"})
	inspection, err := Inspect(root)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if len(inspection.Warnings) != 1 || !strings.Contains(inspection.Warnings[0], "_P suffix") {
		t.Fatalf("Inspect() warnings = %v, want _P warning", inspection.Warnings)
	}
}

func TestInspectRejectsSourceWithoutPackages(t *testing.T) {
	t.Parallel()

	root := makeSource(t, map[string]string{"readme.txt": "text"})
	_, err := Inspect(root)
	if err == nil || !strings.Contains(err.Error(), "contains no") {
		t.Fatalf("Inspect() error = %v, want no packages", err)
	}
}

func makeSource(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for name, contents := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create source parent: %v", err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write source file: %v", err)
		}
	}
	return root
}
