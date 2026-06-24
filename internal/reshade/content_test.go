package reshade

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInventoryUserContentHashesInternalFilesAndDoesNotTraverseExternalPaths(t *testing.T) {
	t.Parallel()
	gameRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "bin")
	if err := os.MkdirAll(filepath.Join(targetPath, "reshade-shaders", "Shaders"), 0o755); err != nil {
		t.Fatal(err)
	}
	presetPath := filepath.Join(targetPath, "Custom.ini")
	effectPath := filepath.Join(targetPath, "reshade-shaders", "Shaders", "Effect.fx")
	externalPath := filepath.Join(t.TempDir(), "ExternalTextures")
	if err := os.MkdirAll(externalPath, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, contents := range map[string]string{
		presetPath: "preset",
		effectPath: "effect",
		filepath.Join(externalPath, "Texture.png"): "texture",
	} {
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	config := strings.Join([]string{
		"[GENERAL]",
		"PresetPath=Custom.ini",
		"EffectSearchPaths=reshade-shaders\\Shaders",
		"TextureSearchPaths=" + externalPath,
	}, "\n")
	if err := os.WriteFile(filepath.Join(targetPath, "ReShade.ini"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	content, warnings, err := inventoryUserContent(gameRoot, targetPath)
	if err != nil {
		t.Fatal(err)
	}
	var foundPreset, foundEffect, foundExternal bool
	for _, item := range content {
		switch {
		case strings.EqualFold(item.Path, filepath.Join("bin", "Custom.ini")):
			foundPreset = item.SHA256 != "" && item.Exists
		case strings.EqualFold(item.Path, filepath.Join("bin", "reshade-shaders", "Shaders", "Effect.fx")):
			foundEffect = item.SHA256 != "" && item.Exists
		case strings.EqualFold(item.Path, externalPath):
			foundExternal = item.External && item.InventoryOnly && item.SHA256 == ""
		}
	}
	if !foundPreset || !foundEffect || !foundExternal || len(warnings) == 0 {
		t.Fatalf("content = %+v, warnings = %+v", content, warnings)
	}
}

func TestInventoryUserContentIgnoresMissingDefaultPaths(t *testing.T) {
	t.Parallel()
	gameRoot := t.TempDir()
	targetPath := filepath.Join(gameRoot, "bin")
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		t.Fatal(err)
	}
	content, _, err := inventoryUserContent(gameRoot, targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) != 0 {
		t.Fatalf("expected no inventoried content without ReShade.ini, got %+v", content)
	}
}

func TestDetectUserContentDriftIsSeparateFromManagedDrift(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "Preset.ini")
	if err := os.WriteFile(path, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	drift, err := detectUserContentDrift(root, Manifest{
		Version: ManifestVersion,
		UserContent: []UserContent{
			{
				Path:      "Preset.ini",
				Role:      PathRolePreset,
				SHA256:    strings.Repeat("0", 64),
				SizeBytes: 1,
				Exists:    true,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) != 1 || drift[0].Role != PathRolePreset || drift[0].Missing {
		t.Fatalf("drift = %+v", drift)
	}
}
