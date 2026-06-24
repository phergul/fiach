package reshade

import (
	"archive/zip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
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

func TestParseContentCatalogueSkipsAddonsWithoutDownloadURL(t *testing.T) {
	t.Parallel()
	catalogue, err := parseContentCatalogue(
		`[00] Enabled=1 Required=1 PackageName=Standard effects PackageDescription=Utility InstallPath=.\reshade-shaders\Shaders TextureInstallPath=.\reshade-shaders\Textures DownloadUrl=https://github.com/crosire/reshade-shaders/archive/slim.zip RepositoryUrl=https://github.com/crosire/reshade-shaders EffectFiles=DisplayDepth.fx`,
		`[14] PackageName=Unavailable add-on PackageDescription=No release asset RepositoryUrl=https://github.com/crosire/reshade
[15] PackageName=Available add-on PackageDescription=Has release asset DownloadUrl64=https://github.com/crosire/reshade-docs/releases/latest/download/a.addon64 RepositoryUrl=https://github.com/crosire/reshade`,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalogue.Addons) != 1 || catalogue.Addons[0].ID != "15" {
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
	manager := NewManager(store, ManagerOptions{
		DataDir:       dataDir,
		VerifyApplied: func(string, Preview) error { return nil },
	})
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

func TestInstallPreviewCombinesLifecycleAndContent(t *testing.T) {
	t.Parallel()

	root, request := newReShadeRequest(t)
	request.Content = ContentRequest{
		EffectPackages: []EffectPackageSelection{{ID: "00"}},
	}
	runtimeStage := filepath.Join(t.TempDir(), "dxgi.dll")
	if err := os.WriteFile(runtimeStage, []byte("runtime"), 0o644); err != nil {
		t.Fatal(err)
	}
	runtimeHash, runtimeSize, err := fileops.FileIntegrity(runtimeStage)
	if err != nil {
		t.Fatal(err)
	}
	dataDir := t.TempDir()
	writeContentCache(t, dataDir, "https://github.com/crosire/reshade-shaders/archive/slim.zip")
	writeArchiveCache(t, dataDir, "https://github.com/crosire/reshade-shaders/archive/slim.zip", map[string]string{
		"reshade-shaders-slim/Shaders/Blending.fxh":    "blend",
		"reshade-shaders-slim/Shaders/ReShade.fxh":     "common",
		"reshade-shaders-slim/Shaders/DisplayDepth.fx": "shader",
		"reshade-shaders-slim/Textures/lut.png":        "texture",
	})
	manager := NewManager(newMemoryReShadeStore(), ManagerOptions{
		DataDir: dataDir,
		Planner: PlannerFunc(func(
			context.Context,
			string,
			Request,
			*dbtypes.ReShadeTarget,
		) (Preview, error) {
			return Preview{
				Operations: []Operation{{
					Type:       "copy",
					SourcePath: runtimeStage,
					TargetPath: filepath.Join(root, request.ProxyFilename),
					SHA256:     runtimeHash,
					SizeBytes:  runtimeSize,
				}},
				PathImpacts: []PathImpact{pathImpact(
					request.ProxyFilename,
					PathRoleRuntime,
					"replace",
					OwnershipManaged,
					false,
					false,
				)},
				DesiredTarget: &TargetState{
					RuntimeVersion:   "6",
					ManagementOrigin: "installed",
					Manifest: Manifest{
						Version: ManifestVersion,
						Files: []ManagedFile{{
							RelativePath: request.ProxyFilename,
							SHA256:       runtimeHash,
							SizeBytes:    runtimeSize,
							Ownership:    OwnershipManaged,
							Role:         PathRoleRuntime,
						}},
						VariantProvenance: VariantProvenanceVerified,
					},
				},
			}, nil
		}),
		VerifyApplied: func(string, Preview) error { return nil },
	})

	preview, err := manager.Preview(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	shaderPath := filepath.Join(root, "reshade-shaders", "Shaders", "DisplayDepth.fx")
	if !preview.CanApply ||
		!hasOperation(preview.Operations, "copy", filepath.Join(root, request.ProxyFilename)) ||
		!hasOperation(preview.Operations, "copy", shaderPath) {
		t.Fatalf("Preview() = %+v", preview)
	}
	if preview.DesiredTarget == nil || !manifestHasSource(preview.DesiredTarget.Manifest, "00") {
		t.Fatalf("desired target = %+v", preview.DesiredTarget)
	}
	repeatedPreview, err := manager.Preview(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if repeatedPreview.PreviewHash != preview.PreviewHash {
		t.Fatalf("repeated preview hash = %q, want %q", repeatedPreview.PreviewHash, preview.PreviewHash)
	}
}

func TestConfigureContentPreviewRemovesDeselectedManagedContent(t *testing.T) {
	t.Parallel()
	root, request := newManagedContentRequest(t, BuildVariantStandard)
	managedPath := filepath.Join(root, "reshade-shaders", "Shaders", "DisplayDepth.fx")
	if err := os.MkdirAll(filepath.Dir(managedPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managedPath, []byte("managed"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, size, _ := fileops.FileIntegrity(managedPath)
	store := newMemoryReShadeStore()
	store.targets[store.key(1, ".")] = managedContentTarget(t, root, request, []ManagedFile{
		contentManagedFile("reshade-shaders/Shaders/DisplayDepth.fx", hash, size, PathRoleEffects, "00"),
	})
	dataDir := t.TempDir()
	writeContentCache(t, dataDir, "https://github.com/crosire/reshade-shaders/archive/slim.zip")
	manager := NewManager(store, ManagerOptions{
		DataDir:       dataDir,
		VerifyApplied: func(string, Preview) error { return nil },
	})
	preview, err := manager.Preview(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.CanApply || !hasOperation(preview.Operations, "delete", managedPath) {
		t.Fatalf("Preview() = %+v", preview)
	}
	if preview.DesiredTarget == nil || manifestHasSource(preview.DesiredTarget.Manifest, "00") {
		t.Fatalf("desired manifest still has removed source: %+v", preview.DesiredTarget)
	}
}

func TestUninstallRemovesManagedContentAndPreservesUserContent(t *testing.T) {
	t.Parallel()
	root, request := newReShadeRequest(t)
	request.Action = ActionUninstall
	runtimePath := filepath.Join(root, request.ProxyFilename)
	managedContentPath := filepath.Join(root, "reshade-shaders", "Shaders", "DisplayDepth.fx")
	userPresetPath := filepath.Join(root, "Preset.ini")
	for path, contents := range map[string]string{
		runtimePath:        "runtime",
		managedContentPath: "managed shader",
		userPresetPath:     "user preset",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runtimeHash, runtimeSize, _ := fileops.FileIntegrity(runtimePath)
	contentHash, contentSize, _ := fileops.FileIntegrity(managedContentPath)
	store := newMemoryReShadeStore()
	store.targets[store.key(1, ".")] = managedContentTarget(t, root, request, []ManagedFile{
		{
			RelativePath: request.ProxyFilename,
			SHA256:       runtimeHash,
			SizeBytes:    runtimeSize,
			Ownership:    OwnershipManaged,
			Role:         PathRoleRuntime,
		},
		contentManagedFile("reshade-shaders/Shaders/DisplayDepth.fx", contentHash, contentSize, PathRoleEffects, "00"),
		{
			RelativePath: "Preset.ini",
			SHA256:       "user",
			SizeBytes:    9,
			Ownership:    OwnershipUser,
			Role:         PathRolePreset,
		},
	})
	manager := NewManager(store, ManagerOptions{
		DataDir: t.TempDir(),
		Planner: NewInstallerPlanner(InstallerPlannerOptions{
			InspectArchitecture: func(string) (Architecture, error) {
				return ArchitectureX64, nil
			},
		}),
		VerifyApplied: func(string, Preview) error { return nil },
	})
	preview, err := manager.Preview(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if !hasOperation(preview.Operations, "delete", managedContentPath) ||
		hasOperation(preview.Operations, "delete", userPresetPath) {
		t.Fatalf("uninstall operations = %+v", preview.Operations)
	}
	if _, err := manager.Apply(context.Background(), root, request, preview.PreviewHash); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(managedContentPath); !os.IsNotExist(err) {
		t.Fatalf("managed content still exists or stat failed: %v", err)
	}
	if _, err := os.Stat(userPresetPath); err != nil {
		t.Fatalf("user content was not preserved: %v", err)
	}
}

func TestConfigureContentPreviewSelectsAddonFromArchitectureArchive(t *testing.T) {
	t.Parallel()
	root, request := newManagedContentRequest(t, BuildVariantAddon)
	request.Content = ContentRequest{Addons: []AddonSelection{{ID: "00"}}}
	store := newMemoryReShadeStore()
	store.targets[store.key(1, ".")] = managedContentTarget(t, root, request, nil)
	dataDir := t.TempDir()
	effectURL := "https://github.com/crosire/reshade-shaders/archive/slim.zip"
	addonURL := "https://github.com/crosire/reshade-docs/releases/latest/download/addon.zip"
	writeContentCacheWithAddon(t, dataDir, effectURL, addonURL)
	writeArchiveCacheNamed(t, dataDir, addonURL, "addon.zip", map[string]string{
		"release/Test.addon32": "x86",
		"release/Test.addon64": "x64",
	})
	manager := NewManager(store, ManagerOptions{DataDir: dataDir})
	preview, err := manager.Preview(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.CanApply || !hasOperation(preview.Operations, "copy", filepath.Join(root, "Addons", "Test.addon64")) {
		t.Fatalf("Preview() = %+v", preview)
	}
}

func TestConfigureContentPreviewKeepsSharedFileUntilAllSourcesRemoved(t *testing.T) {
	t.Parallel()
	root, request := newManagedContentRequest(t, BuildVariantStandard)
	request.Content = ContentRequest{EffectPackages: []EffectPackageSelection{{ID: "01"}}}
	sharedPath := filepath.Join(root, "reshade-shaders", "Shaders", "Shared.fx")
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sharedPath, []byte("shared"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, size, _ := fileops.FileIntegrity(sharedPath)
	store := newMemoryReShadeStore()
	store.targets[store.key(1, ".")] = managedContentTarget(t, root, request, []ManagedFile{
		{
			RelativePath: "reshade-shaders/Shaders/Shared.fx",
			SHA256:       hash,
			SizeBytes:    size,
			Ownership:    OwnershipManaged,
			Role:         PathRoleEffects,
			Sources: []ContentSource{
				{Kind: ContentSourceEffectPackage, ID: "00", Shared: true},
				{Kind: ContentSourceEffectPackage, ID: "01", Shared: true},
			},
		},
	})
	dataDir := t.TempDir()
	writeContentCacheWithTwoEffects(t, dataDir)
	writeArchiveCacheNamed(t, dataDir, "https://github.com/crosire/reshade-shaders/archive/other.zip", "other.zip", map[string]string{
		"pkg/Shaders/Other.fx": "other",
	})
	manager := NewManager(store, ManagerOptions{
		DataDir:       dataDir,
		VerifyApplied: func(string, Preview) error { return nil },
	})
	preview, err := manager.Preview(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if hasOperation(preview.Operations, "delete", sharedPath) {
		t.Fatalf("shared file was deleted while a selected source remains: %+v", preview.Operations)
	}
}

func TestContentDriftBlocksAndBackupAndContinueArchivesChangedFile(t *testing.T) {
	t.Parallel()
	root, request := newManagedContentRequest(t, BuildVariantStandard)
	managedPath := filepath.Join(root, "reshade-shaders", "Shaders", "DisplayDepth.fx")
	if err := os.MkdirAll(filepath.Dir(managedPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managedPath, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, size, _ := fileops.FileIntegrity(managedPath)
	if err := os.WriteFile(managedPath, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := newMemoryReShadeStore()
	store.targets[store.key(1, ".")] = managedContentTarget(t, root, request, []ManagedFile{
		contentManagedFile("reshade-shaders/Shaders/DisplayDepth.fx", hash, size, PathRoleEffects, "00"),
	})
	dataDir := t.TempDir()
	writeContentCache(t, dataDir, "https://github.com/crosire/reshade-shaders/archive/slim.zip")
	manager := NewManager(store, ManagerOptions{DataDir: dataDir})
	manager.verifyApplied = func(string, Preview) error { return nil }
	preview, err := manager.Preview(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if preview.CanApply || len(preview.Drift) != 1 {
		t.Fatalf("drift preview = %+v", preview)
	}
	request.BackupAndContinue = true
	preview, err = manager.Preview(context.Background(), root, request)
	if err != nil || !preview.CanApply {
		t.Fatalf("backup-and-continue preview = %+v, %v", preview, err)
	}
	if _, err := manager.Apply(context.Background(), root, request, preview.PreviewHash); err != nil {
		t.Fatal(err)
	}
	if !archiveContainsFile(t, filepath.Join(dataDir, "archives", "drift"), "DisplayDepth.fx") {
		t.Fatal("changed content drift was not archived")
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
	writeContentCacheWithAddon(t, dataDir, downloadURL, "https://github.com/crosire/reshade-docs/releases/latest/download/a.addon64")
}

func writeContentCacheWithAddon(t *testing.T, dataDir string, downloadURL string, addonURL string) {
	t.Helper()
	cachePath := filepath.Join(dataDir, "cache", "content-catalogue.json")
	if err := writeContentCatalogueCache(cachePath, contentCatalogueCache{
		EffectContents: `[00] PackageName=Standard effects PackageDescription=Utility InstallPath=.\reshade-shaders\Shaders TextureInstallPath=.\reshade-shaders\Textures DownloadUrl=` + downloadURL + ` RepositoryUrl=https://github.com/crosire/reshade-shaders EffectFiles=DisplayDepth.fx`,
		AddonContents:  `[00] PackageName=Swap chain override PackageDescription=Addon DownloadUrl64=` + addonURL + ` RepositoryUrl=https://github.com/crosire/reshade`,
	}); err != nil {
		t.Fatal(err)
	}
}

func writeContentCacheWithTwoEffects(t *testing.T, dataDir string) {
	t.Helper()
	cachePath := filepath.Join(dataDir, "cache", "content-catalogue.json")
	if err := writeContentCatalogueCache(cachePath, contentCatalogueCache{
		EffectContents: `[00] PackageName=Standard effects PackageDescription=Utility InstallPath=.\reshade-shaders\Shaders DownloadUrl=https://github.com/crosire/reshade-shaders/archive/slim.zip RepositoryUrl=https://github.com/crosire/reshade-shaders EffectFiles=Shared.fx
[01] PackageName=Other effects PackageDescription=Other InstallPath=.\reshade-shaders\Shaders DownloadUrl=https://github.com/crosire/reshade-shaders/archive/other.zip RepositoryUrl=https://github.com/crosire/reshade-shaders EffectFiles=Other.fx`,
		AddonContents: `[00] PackageName=Swap chain override PackageDescription=Addon DownloadUrl64=https://github.com/crosire/reshade-docs/releases/latest/download/a.addon64 RepositoryUrl=https://github.com/crosire/reshade`,
	}); err != nil {
		t.Fatal(err)
	}
}

func writeArchiveCache(t *testing.T, dataDir string, rawURL string, files map[string]string) {
	t.Helper()
	writeArchiveCacheNamed(t, dataDir, rawURL, "slim.zip", files)
}

func writeArchiveCacheNamed(t *testing.T, dataDir string, rawURL string, filename string, files map[string]string) {
	t.Helper()
	name := hashBytes([]byte(rawURL))[:16] + "-" + filename
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

func newManagedContentRequest(t *testing.T, variant BuildVariant) (string, Request) {
	t.Helper()
	root, request := newReShadeRequest(t)
	request.Action = ActionConfigureContent
	request.BuildVariant = variant
	if variant == BuildVariantAddon {
		request.SinglePlayerAcknowledged = true
		request.AntiCheatRiskAcknowledged = true
	}
	if err := os.WriteFile(filepath.Join(root, "ReShade.ini"), []byte("[GENERAL]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runtimePath := filepath.Join(root, request.ProxyFilename)
	if err := os.WriteFile(runtimePath, []byte("runtime"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, request
}

func managedContentTarget(t *testing.T, root string, request Request, extraFiles []ManagedFile) dbtypes.ReShadeTarget {
	t.Helper()
	runtimePath := filepath.Join(root, request.ProxyFilename)
	runtimeHash, runtimeSize, err := fileops.FileIntegrity(runtimePath)
	if err != nil {
		t.Fatal(err)
	}
	files := []ManagedFile{{
		RelativePath: request.ProxyFilename,
		SHA256:       runtimeHash,
		SizeBytes:    runtimeSize,
		Ownership:    OwnershipManaged,
		Role:         PathRoleRuntime,
	}}
	files = append(files, extraFiles...)
	digest := "test-digest"
	assetName := "ReShade_Setup_6.exe"
	installerURL := "https://reshade.me/downloads/ReShade_Setup_6.exe"
	installerSize := int64(12)
	return dbtypes.ReShadeTarget{
		ID:                     1,
		GameID:                 request.GameID,
		TargetRelativePath:     request.TargetRelativePath,
		ExecutableRelativePath: request.ExecutableRelativePath,
		RenderingAPI:           string(request.RenderingAPI),
		ProxyFilename:          request.ProxyFilename,
		Architecture:           string(request.Architecture),
		BuildVariant:           string(request.BuildVariant),
		RuntimeVersion:         "6",
		InstallerAssetName:     &assetName,
		InstallerURL:           &installerURL,
		InstallerDigest:        &digest,
		InstallerSize:          &installerSize,
		ManagementOrigin:       "installed",
		Status:                 "managed",
		ManifestJSON: encodeTestManifest(t, Manifest{
			Version:           ManifestVersion,
			Files:             files,
			VariantProvenance: VariantProvenanceVerified,
		}),
	}
}

func contentManagedFile(path string, hash string, size int64, role PathRole, sourceID string) ManagedFile {
	return ManagedFile{
		RelativePath: path,
		SHA256:       hash,
		SizeBytes:    size,
		Ownership:    OwnershipManaged,
		Role:         role,
		Sources: []ContentSource{{
			Kind: ContentSourceEffectPackage,
			ID:   sourceID,
		}},
	}
}

func hasOperation(operations []Operation, operationType string, targetPath string) bool {
	return slices.ContainsFunc(operations, func(operation Operation) bool {
		return operation.Type == operationType && strings.EqualFold(
			filepath.Clean(operation.TargetPath),
			filepath.Clean(targetPath),
		)
	})
}

func archiveContainsFile(t *testing.T, root string, name string) bool {
	t.Helper()
	found := false
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if strings.EqualFold(entry.Name(), name) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}
