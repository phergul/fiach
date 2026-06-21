package reshade

import (
	"archive/zip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestParseContentCatalogueReadsEffectsAndAddons(t *testing.T) {
	t.Parallel()
	catalogue, err := parseContentCatalogue(
		`[00] Enabled=1 Required=1 PackageName=Standard effects PackageDescription=Utility InstallPath=.\reshade-shaders\Shaders TextureInstallPath=.\reshade-shaders\Textures DownloadUrl=https://github.com/crosire/reshade-shaders/archive/slim.zip RepositoryUrl=https://github.com/crosire/reshade-shaders EffectFiles=DisplayDepth.fx,UIMask.fx DenyEffectFiles=Template.fx`,
		`[00] PackageName=Swap chain override PackageDescription=Addon DownloadUrl32=https://github.com/crosire/reshade-docs/releases/latest/download/a.addon32 DownloadUrl64=https://github.com/crosire/reshade-docs/releases/latest/download/a.addon64 RepositoryUrl=https://github.com/crosire/reshade`,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalogue.Effects) != 1 || !catalogue.Effects[0].Required ||
		catalogue.Effects[0].EffectFiles[0] != "DisplayDepth.fx" ||
		catalogue.Effects[0].DenyEffectFiles[0] != "Template.fx" {
		t.Fatalf("effects = %+v", catalogue.Effects)
	}
	if len(catalogue.Addons) != 1 || catalogue.Addons[0].DownloadURL64 == "" {
		t.Fatalf("addons = %+v", catalogue.Addons)
	}
}

func TestInspectPresetRecommendsMatchingPackages(t *testing.T) {
	t.Parallel()
	presetPath := filepath.Join(t.TempDir(), "Preset.ini")
	if err := os.WriteFile(presetPath, []byte("Techniques=MXAO@qUINT_mxao.fx,Depth@DisplayDepth.fx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := InspectPreset(presetPath, ContentCatalogue{Effects: []EffectPackage{
		{
			ID:          "00",
			Name:        "Standard effects",
			EffectFiles: []string{"DisplayDepth.fx"},
		},
		{
			ID:          "09",
			Name:        "qUINT",
			EffectFiles: []string{"qUINT_mxao.fx"},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Recommendations) != 2 || len(result.MissingEffects) != 0 {
		t.Fatalf("InspectPreset() = %+v", result)
	}
}

func TestConfigureContentPreviewUsesCachedCatalogueAndArchive(t *testing.T) {
	t.Parallel()
	root, request := newReShadeRequest(t)
	request.Action = ActionConfigureContent
	request.Content = ContentRequest{
		EffectPackages: []EffectPackageSelection{{ID: "00"}},
	}
	if err := os.WriteFile(filepath.Join(root, "ReShade.ini"), []byte("[GENERAL]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runtimePath := filepath.Join(root, request.ProxyFilename)
	if err := os.WriteFile(runtimePath, []byte("runtime"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, size, err := fileops.FileIntegrity(runtimePath)
	if err != nil {
		t.Fatal(err)
	}
	manifestJSON := encodeTestManifest(t, Manifest{
		Version: ManifestVersion,
		Files: []ManagedFile{{
			RelativePath: request.ProxyFilename,
			SHA256:       hash,
			SizeBytes:    size,
			Ownership:    OwnershipManaged,
			Role:         PathRoleRuntime,
		}},
		VariantProvenance: VariantProvenanceVerified,
	})
	store := newMemoryReShadeStore()
	store.targets[store.key(1, ".")] = dbtypes.ReShadeTarget{
		ID:                     1,
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           "d3d11",
		ProxyFilename:          "dxgi.dll",
		Architecture:           "x64",
		BuildVariant:           "standard",
		RuntimeVersion:         "6",
		ManagementOrigin:       "installed",
		Status:                 "managed",
		ManifestJSON:           manifestJSON,
	}
	dataDir := t.TempDir()
	downloadURL := "https://github.com/crosire/reshade-shaders/archive/slim.zip"
	writeContentCache(t, dataDir, downloadURL)
	writeArchiveCache(t, dataDir, downloadURL, map[string]string{
		"reshade-shaders-slim/Shaders/DisplayDepth.fx": "shader",
		"reshade-shaders-slim/Textures/UIMask.png":     "texture",
	})
	manager := NewManager(store, ManagerOptions{DataDir: dataDir})
	preview, err := manager.Preview(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.CanApply || len(preview.Conflicts) != 0 {
		t.Fatalf("Preview() = %+v", preview)
	}
	if len(preview.Operations) != 3 {
		t.Fatalf("operations = %+v", preview.Operations)
	}
	if preview.DesiredTarget == nil || !manifestHasSource(preview.DesiredTarget.Manifest, "00") {
		t.Fatalf("manifest = %+v", preview.DesiredTarget)
	}
}

func TestConfigureContentPreviewConflictsOnUserOwnedFile(t *testing.T) {
	t.Parallel()
	root, request := newReShadeRequest(t)
	request.Action = ActionConfigureContent
	request.Content = ContentRequest{
		EffectPackages: []EffectPackageSelection{{ID: "00"}},
	}
	if err := os.MkdirAll(filepath.Join(root, "reshade-shaders", "Shaders"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "reshade-shaders", "Shaders", "DisplayDepth.fx"), []byte("user"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ReShade.ini"), []byte("[GENERAL]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runtimePath := filepath.Join(root, request.ProxyFilename)
	if err := os.WriteFile(runtimePath, []byte("runtime"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, size, _ := fileops.FileIntegrity(runtimePath)
	store := newMemoryReShadeStore()
	store.targets[store.key(1, ".")] = dbtypes.ReShadeTarget{
		ID:                     1,
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           "d3d11",
		ProxyFilename:          "dxgi.dll",
		Architecture:           "x64",
		BuildVariant:           "standard",
		RuntimeVersion:         "6",
		ManagementOrigin:       "installed",
		Status:                 "managed",
		ManifestJSON: encodeTestManifest(t, Manifest{
			Version: ManifestVersion,
			Files: []ManagedFile{{
				RelativePath: request.ProxyFilename,
				SHA256:       hash,
				SizeBytes:    size,
				Ownership:    OwnershipManaged,
			}},
			VariantProvenance: VariantProvenanceVerified,
		}),
	}
	dataDir := t.TempDir()
	downloadURL := "https://github.com/crosire/reshade-shaders/archive/slim.zip"
	writeContentCache(t, dataDir, downloadURL)
	writeArchiveCache(t, dataDir, downloadURL, map[string]string{
		"reshade-shaders-slim/Shaders/DisplayDepth.fx": "shader",
	})
	manager := NewManager(store, ManagerOptions{DataDir: dataDir})
	preview, err := manager.Preview(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if preview.CanApply || !strings.Contains(strings.Join(preview.Conflicts, "\n"), "user-owned") {
		t.Fatalf("Preview() = %+v", preview)
	}
}

func encodeTestManifest(t *testing.T, manifest Manifest) string {
	t.Helper()
	contents, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}

func writeContentCache(t *testing.T, dataDir string, downloadURL string) {
	t.Helper()
	cachePath := filepath.Join(dataDir, "cache", "content-catalogue.json")
	if err := writeContentCatalogueCache(cachePath, contentCatalogueCache{
		EffectContents: `[00] PackageName=Standard effects PackageDescription=Utility InstallPath=.\reshade-shaders\Shaders TextureInstallPath=.\reshade-shaders\Textures DownloadUrl=` + downloadURL + ` RepositoryUrl=https://github.com/crosire/reshade-shaders EffectFiles=DisplayDepth.fx`,
		AddonContents:  `[00] PackageName=Swap chain override PackageDescription=Addon DownloadUrl64=https://github.com/crosire/reshade-docs/releases/latest/download/a.addon64 RepositoryUrl=https://github.com/crosire/reshade`,
	}); err != nil {
		t.Fatal(err)
	}
}

func writeArchiveCache(t *testing.T, dataDir string, rawURL string, files map[string]string) {
	t.Helper()
	name := hashBytes([]byte(rawURL))[:16] + "-slim.zip"
	path := filepath.Join(dataDir, "cache", "content-archives", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	output, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(output)
	for name, contents := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
}

func manifestHasSource(manifest Manifest, id string) bool {
	for _, file := range manifest.Files {
		for _, source := range fileSources(file) {
			if source.ID == id {
				return true
			}
		}
	}
	return false
}
