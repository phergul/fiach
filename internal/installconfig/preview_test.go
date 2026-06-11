package installconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBuildPreviewMapsGenericCopyTargetPaths(t *testing.T) {
	t.Parallel()

	sourcePath := makePreviewSource(t, map[string]string{
		"Data/SkyUI.esp": "plugin",
		"Data/.DS_Store": "metadata",
		".DS_Store":      "metadata",
		"readme.txt":     "hello",
	})

	preview, err := BuildPreview(PreviewInput{
		SourcePath:         sourcePath,
		StrategyType:       StrategyTypeGenericCopy,
		TargetRelativePath: "Mods/SkyUI",
		FileCap:            100,
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}

	wantPaths := []string{
		"Mods/SkyUI/Data/SkyUI.esp",
		"Mods/SkyUI/readme.txt",
	}
	if preview.StrategyType != StrategyTypeGenericCopy || preview.TargetBase != TargetBaseGameRoot || preview.TargetRelativePath != "Mods/SkyUI" || preview.TotalFileCount != 2 || preview.IsCapped {
		t.Fatalf("BuildPreview() = %+v, want generic copy preview with 2 uncapped files", preview)
	}
	if !reflect.DeepEqual(preview.TargetFilePaths, wantPaths) {
		t.Fatalf("TargetFilePaths = %+v, want %+v", preview.TargetFilePaths, wantPaths)
	}
}

func TestBuildPreviewUsesGameRootTarget(t *testing.T) {
	t.Parallel()

	sourcePath := makePreviewSource(t, map[string]string{
		"Data/SkyUI.esp": "plugin",
	})

	preview, err := BuildPreview(PreviewInput{
		SourcePath:         sourcePath,
		StrategyType:       StrategyTypeGenericCopy,
		TargetRelativePath: ".",
		FileCap:            100,
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}

	if preview.TargetDisplayPath != "Game root" || !reflect.DeepEqual(preview.TargetFilePaths, []string{"Data/SkyUI.esp"}) {
		t.Fatalf("BuildPreview() = %+v, want game root target path", preview)
	}
}

func TestBuildPreviewCapsTargetPathList(t *testing.T) {
	t.Parallel()

	sourcePath := makePreviewSource(t, map[string]string{
		"a.txt": "a",
		"b.txt": "b",
		"c.txt": "c",
	})

	preview, err := BuildPreview(PreviewInput{
		SourcePath:         sourcePath,
		StrategyType:       StrategyTypeGenericCopy,
		TargetRelativePath: "Data",
		FileCap:            2,
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}

	if !preview.IsCapped || preview.Cap != 2 || preview.TotalFileCount != 3 || len(preview.TargetFilePaths) != 2 || len(preview.Warnings) == 0 {
		t.Fatalf("BuildPreview() = %+v, want capped preview", preview)
	}
}

func TestBuildPreviewMapsUnrealPackageFilesFlat(t *testing.T) {
	t.Parallel()

	sourcePath := makePreviewSource(t, map[string]string{
		"Nested/Example_P.pak": "pak",
		"docs/readme.txt":      "ignored",
	})

	preview, err := BuildPreview(PreviewInput{
		SourcePath:         sourcePath,
		StrategyType:       StrategyTypeUnrealPak,
		TargetRelativePath: "Example/Content/Paks/~mods",
		FileCap:            100,
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}

	wantPaths := []string{"Example/Content/Paks/~mods/Example_P.pak"}
	if preview.TotalFileCount != 1 || preview.TotalDirectoryCount != 0 || !reflect.DeepEqual(preview.TargetFilePaths, wantPaths) {
		t.Fatalf("BuildPreview() = %+v, want flat Unreal package preview", preview)
	}
	if len(preview.Warnings) != 1 || !strings.Contains(preview.Warnings[0], "Ignored 1") {
		t.Fatalf("BuildPreview() warnings = %v, want ignored-file warning", preview.Warnings)
	}
}

func makePreviewSource(t *testing.T, files map[string]string) string {
	t.Helper()

	sourcePath := t.TempDir()
	for name, contents := range files {
		path := filepath.Join(sourcePath, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create preview source parent: %v", err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write preview source file: %v", err)
		}
	}

	return sourcePath
}
