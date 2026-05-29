package reshade

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestScanDetectsReShadeWithIniSupport(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "bin")
	writeReShadeTestFile(t, filepath.Join(target, "Game.exe"))
	writeReShadeTestFile(t, filepath.Join(target, "dxgi.dll"))
	writeReShadeTestFile(t, filepath.Join(target, "ReShade.ini"))

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	assertReShadeTargets(t, result, []Target{
		{
			Path:        target,
			Executables: []string{filepath.Join(target, "Game.exe")},
		},
	})
}

func TestScanDetectsReShadeWithShadersSupport(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "x64")
	writeReShadeTestFile(t, filepath.Join(target, "Game.exe"))
	writeReShadeTestFile(t, filepath.Join(target, "d3d11.dll"))
	mkdirReShadeTestDir(t, filepath.Join(target, "reshade-shaders"))

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	assertReShadeTargets(t, result, []Target{
		{
			Path:        target,
			Executables: []string{filepath.Join(target, "Game.exe")},
		},
	})
}

func TestScanRequiresSupportEvidence(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "game")
	writeReShadeTestFile(t, filepath.Join(target, "Game.exe"))
	writeReShadeTestFile(t, filepath.Join(target, "dxgi.dll"))

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	assertReShadeTargets(t, result, nil)
}

func TestScanRequiresKnownDLL(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "game")
	writeReShadeTestFile(t, filepath.Join(target, "Game.exe"))
	writeReShadeTestFile(t, filepath.Join(target, "ReShade.ini"))

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	assertReShadeTargets(t, result, nil)
}

func TestScanIgnoresPresetIniAsSupportEvidence(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "game")
	writeReShadeTestFile(t, filepath.Join(target, "Game.exe"))
	writeReShadeTestFile(t, filepath.Join(target, "dxgi.dll"))
	writeReShadeTestFile(t, filepath.Join(target, "Preset.ini"))

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	assertReShadeTargets(t, result, nil)
}

func TestScanMatchesMarkersCaseInsensitively(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "GAME")
	writeReShadeTestFile(t, filepath.Join(target, "GAME.EXE"))
	writeReShadeTestFile(t, filepath.Join(target, "DXGI.DLL"))
	writeReShadeTestFile(t, filepath.Join(target, "reshade.ini"))

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	assertReShadeTargets(t, result, []Target{
		{
			Path:        target,
			Executables: []string{filepath.Join(target, "GAME.EXE")},
		},
	})
}

func TestScanReturnsCandidateFolderAndExecutables(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "game")
	writeReShadeTestFile(t, filepath.Join(target, "B.exe"))
	writeReShadeTestFile(t, filepath.Join(target, "A.exe"))
	writeReShadeTestFile(t, filepath.Join(target, "opengl32.dll"))
	writeReShadeTestFile(t, filepath.Join(target, "ReShade.ini"))

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	assertReShadeTargets(t, result, []Target{
		{
			Path: target,
			Executables: []string{
				filepath.Join(target, "A.exe"),
				filepath.Join(target, "B.exe"),
			},
		},
	})
}

func assertReShadeTargets(t *testing.T, result Result, want []Target) {
	t.Helper()

	if want == nil {
		want = []Target{}
	}
	if result.Targets == nil {
		result.Targets = []Target{}
	}
	if !reflect.DeepEqual(result.Targets, want) {
		t.Fatalf("Targets = %#v, want %#v", result.Targets, want)
	}
}

func writeReShadeTestFile(t *testing.T, path string) {
	t.Helper()

	mkdirReShadeTestDir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func mkdirReShadeTestDir(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}
