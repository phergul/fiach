package reshade

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/storage/dbtypes"
	"github.com/phergul/fiach/internal/winversion"
)

func TestDirectXPlannerInstallUsesPreparedRuntimeAndPreservesExistingConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	executable := filepath.Join(root, "Game.exe")
	config := filepath.Join(root, "ReShade.ini")
	if err := os.WriteFile(executable, []byte("game"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, []byte("[GENERAL]\nPresetPath=Custom.ini\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	planner, prepared := testDirectXPlanner(t)
	request := Request{
		Action:                 ActionInstall,
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           RenderingAPID3D11,
		ProxyFilename:          "dxgi.dll",
		Architecture:           ArchitectureX64,
		BuildVariant:           BuildVariantStandard,
	}
	preview, err := planner.Plan(context.Background(), root, request, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Operations) != 2 {
		t.Fatalf("operations = %+v", preview.Operations)
	}
	for _, operation := range preview.Operations {
		if strings.EqualFold(filepath.Base(operation.TargetPath), "ReShade.ini") {
			t.Fatalf("existing ReShade.ini was scheduled for replacement: %+v", operation)
		}
	}
	if preview.DesiredTarget == nil ||
		preview.DesiredTarget.Manifest.VariantProvenance != VariantProvenanceVerified ||
		preview.DesiredTarget.RuntimeVersion != "6.7.3" {
		t.Fatalf("desired target = %+v", preview.DesiredTarget)
	}
	if preview.Operations[0].SourcePath != prepared["dxgi.dll"].Path {
		t.Fatalf("runtime source = %q", preview.Operations[0].SourcePath)
	}
}

func TestDirectXPlannerAllowsInstallerDefaultProxyOutputForDXGISelection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Game.exe"), []byte("game"), 0o644); err != nil {
		t.Fatal(err)
	}
	stagedRuntime := filepath.Join(t.TempDir(), "d3d11.dll")
	if err := os.WriteFile(stagedRuntime, []byte("runtime"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, size, err := fileops.FileIntegrity(stagedRuntime)
	if err != nil {
		t.Fatal(err)
	}
	artifact := InstallerArtifact{
		InstallerRelease: InstallerRelease{
			Version:   "6.7.3",
			Variant:   InstallerVariantStandard,
			AssetName: "ReShade_Setup_6.7.3.exe",
			URL:       "https://reshade.me/downloads/ReShade_Setup_6.7.3.exe",
		},
		Path:      filepath.Join(root, "installer.exe"),
		SizeBytes: 100,
		SHA256:    strings.Repeat("a", 64),
	}
	planner := NewDirectXPlanner(DirectXPlannerOptions{
		ResolveInstaller: func(
			context.Context,
			InstallerVariant,
			InstallerResolveOptions,
		) (InstallerRelease, error) {
			return artifact.InstallerRelease, nil
		},
		AcquireInstaller: func(
			context.Context,
			InstallerRelease,
			InstallerAcquireOptions,
		) (InstallerArtifact, error) {
			return artifact, nil
		},
		PrepareSetup: func(
			_ context.Context,
			request SetupRequest,
			_ SetupRunnerOptions,
		) (SetupRunResult, error) {
			if !strings.EqualFold(request.ExpectedProxy, "dxgi.dll") ||
				len(request.AllowedProxyOutputRelativePaths) != 1 ||
				!strings.EqualFold(request.AllowedProxyOutputRelativePaths[0], "d3d11.dll") {
				t.Fatalf("setup request = %+v", request)
			}
			return SetupRunResult{
				Prepared: &PreparedSetup{
					Files: []PreparedSetupFile{
						{
							RelativePath: "dxgi.dll",
							Path:         stagedRuntime,
							SHA256:       hash,
							SizeBytes:    size,
						},
					},
				},
			}, nil
		},
		InspectArchitecture: func(string) (Architecture, error) {
			return ArchitectureX64, nil
		},
		ReadMetadata: func(string) (winversion.Metadata, error) {
			return winversion.Metadata{}, nil
		},
	})

	preview, err := planner.Plan(context.Background(), root, Request{
		Action:                 ActionInstall,
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           RenderingAPID3D11,
		ProxyFilename:          "dxgi.dll",
		Architecture:           ArchitectureX64,
		BuildVariant:           BuildVariantStandard,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Operations) != 1 ||
		filepath.Base(preview.Operations[0].TargetPath) != "dxgi.dll" ||
		preview.DesiredTarget == nil ||
		len(preview.DesiredTarget.Manifest.Files) != 1 ||
		preview.DesiredTarget.Manifest.Files[0].RelativePath != "dxgi.dll" {
		t.Fatalf("preview = %+v", preview)
	}
}

func TestDirectXPlannerAdoptRecordsUserDeclaredVariant(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	executable := filepath.Join(root, "Game.exe")
	runtime := filepath.Join(root, "dxgi.dll")
	for _, path := range []string{executable, runtime} {
		if err := os.WriteFile(path, []byte(filepath.Base(path)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	planner := NewDirectXPlanner(DirectXPlannerOptions{
		InspectArchitecture: func(string) (Architecture, error) {
			return ArchitectureX64, nil
		},
		ReadMetadata: func(path string) (winversion.Metadata, error) {
			if strings.EqualFold(filepath.Base(path), "dxgi.dll") {
				return winversion.Metadata{
					ProductName:      "ReShade",
					OriginalFilename: "ReShade64.dll",
					ProductVersion:   "6.7.3",
				}, nil
			}
			return winversion.Metadata{}, nil
		},
	})
	preview, err := planner.Plan(context.Background(), root, Request{
		Action:                 ActionAdopt,
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           RenderingAPID3D11,
		ProxyFilename:          "dxgi.dll",
		Architecture:           ArchitectureX64,
		BuildVariant:           BuildVariantStandard,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if preview.DesiredTarget == nil ||
		preview.DesiredTarget.ManagementOrigin != "adopted" ||
		preview.DesiredTarget.Manifest.VariantProvenance != VariantProvenanceUserDeclared ||
		preview.DesiredTarget.Manifest.HasPreAdoptionRollbackData {
		t.Fatalf("desired target = %+v", preview.DesiredTarget)
	}
	if len(preview.Operations) != 1 || preview.Operations[0].Type != "adopt" {
		t.Fatalf("operations = %+v", preview.Operations)
	}
}

func TestDirectXPlannerRepairUsesRecordedRelease(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	executable := filepath.Join(root, "Game.exe")
	if err := os.WriteFile(executable, []byte("game"), 0o644); err != nil {
		t.Fatal(err)
	}
	planner, _ := testDirectXPlanner(t)
	manifest := `{"version":1,"files":[{"relativePath":"dxgi.dll","sha256":"missing","sizeBytes":7,"ownership":"managed"}],"variantProvenance":"verified"}`
	tag := "v6.7.3"
	asset := "ReShade_Setup_6.7.3.exe"
	url := "https://reshade.me/downloads/ReShade_Setup_6.7.3.exe"
	digest := strings.Repeat("a", 64)
	size := int64(100)
	existing := &dbtypes.ReShadeTarget{
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           "d3d11",
		ProxyFilename:          "dxgi.dll",
		Architecture:           "x64",
		BuildVariant:           "standard",
		RuntimeVersion:         "6.7.3",
		InstallerTag:           &tag,
		InstallerAssetName:     &asset,
		InstallerURL:           &url,
		InstallerDigest:        &digest,
		InstallerSize:          &size,
		ManagementOrigin:       "installed",
		ManifestJSON:           manifest,
	}
	preview, err := planner.Plan(context.Background(), root, Request{
		Action:                 ActionRepair,
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           RenderingAPID3D11,
		ProxyFilename:          "dxgi.dll",
		Architecture:           ArchitectureX64,
		BuildVariant:           BuildVariantStandard,
	}, existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Operations) != 1 || preview.DesiredTarget == nil ||
		preview.DesiredTarget.RuntimeVersion != "6.7.3" {
		t.Fatalf("preview = %+v", preview)
	}
}

func TestDirectXPlannerUpdateCanChangeProxyLayout(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, name := range []string{"Game.exe", "dxgi.dll"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	planner, _ := testDirectXPlanner(t)
	existing := &dbtypes.ReShadeTarget{
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           "d3d11",
		ProxyFilename:          "dxgi.dll",
		Architecture:           "x64",
		BuildVariant:           "standard",
		RuntimeVersion:         "6.6.0",
		ManagementOrigin:       "installed",
		ManifestJSON:           `{"version":1,"files":[{"relativePath":"dxgi.dll","sha256":"old","sizeBytes":8,"ownership":"managed"}],"variantProvenance":"verified"}`,
	}
	preview, err := planner.Plan(context.Background(), root, Request{
		Action:                 ActionUpdate,
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           RenderingAPID3D11,
		ProxyFilename:          "d3d11.dll",
		Architecture:           ArchitectureX64,
		BuildVariant:           BuildVariantStandard,
	}, existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Operations) != 2 ||
		preview.Operations[0].Type != "delete" ||
		filepath.Base(preview.Operations[1].TargetPath) != "d3d11.dll" ||
		len(preview.Warnings) == 0 {
		t.Fatalf("preview = %+v", preview)
	}
}

func TestDirectXPlannerUninstallPreservesUserContent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Game.exe"), []byte("game"), 0o644); err != nil {
		t.Fatal(err)
	}
	planner, _ := testDirectXPlanner(t)
	existing := &dbtypes.ReShadeTarget{
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           "d3d11",
		ProxyFilename:          "dxgi.dll",
		Architecture:           "x64",
		BuildVariant:           "standard",
		RuntimeVersion:         "6.7.3",
		ManagementOrigin:       "adopted",
		ManifestJSON: `{"version":1,"files":[{"relativePath":"dxgi.dll","sha256":"runtime","sizeBytes":7,"ownership":"adopted"}],` +
			`"variantProvenance":"user_declared","userContent":[{"path":"ReShade.ini","role":"configuration","exists":true}]}`,
	}
	preview, err := planner.Plan(context.Background(), root, Request{
		Action:                 ActionUninstall,
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           RenderingAPID3D11,
		ProxyFilename:          "dxgi.dll",
		Architecture:           ArchitectureX64,
		BuildVariant:           BuildVariantStandard,
	}, existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Operations) != 0 {
		t.Fatalf("operations = %+v", preview.Operations)
	}
	if len(preview.PathImpacts) != 2 ||
		preview.PathImpacts[0].Action != "preserve" ||
		preview.PathImpacts[0].Ownership != OwnershipAdopted ||
		preview.PathImpacts[1].Action != "preserve" ||
		preview.PathImpacts[1].Ownership != OwnershipUser {
		t.Fatalf("path impacts = %+v", preview.PathImpacts)
	}
}

func TestDirectXPlannerInstallPreviewOmitsNonExistentDefaultPaths(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Game.exe"), []byte("game"), 0o644); err != nil {
		t.Fatal(err)
	}
	planner, _ := testDirectXPlanner(t)
	preview, err := planner.Plan(context.Background(), root, Request{
		Action:                 ActionInstall,
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           RenderingAPID3D11,
		ProxyFilename:          "dxgi.dll",
		Architecture:           ArchitectureX64,
		BuildVariant:           BuildVariantStandard,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, impact := range preview.PathImpacts {
		if impact.Action != "preserve" {
			continue
		}
		t.Fatalf("unexpected preserve impact on fresh install: %+v", impact)
	}
	createPaths := map[string]bool{}
	for _, impact := range preview.PathImpacts {
		if impact.Action == "create" {
			createPaths[strings.ToLower(impact.Path)] = true
		}
	}
	if !createPaths["reshade.ini"] || !createPaths["reshadepreset.ini"] {
		t.Fatalf("expected create impacts for default config files, got %+v", preview.PathImpacts)
	}
}

func TestDirectXPlannerUninstallDeletesManagedCreatedConfigs(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Game.exe"), []byte("game"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"dxgi.dll", "ReShade.ini", "ReShadePreset.ini"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	planner, _ := testDirectXPlanner(t)
	existing := &dbtypes.ReShadeTarget{
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           "d3d11",
		ProxyFilename:          "dxgi.dll",
		Architecture:           "x64",
		BuildVariant:           "standard",
		RuntimeVersion:         "6.7.3",
		ManagementOrigin:       "installed",
		ManifestJSON: `{"version":1,"files":[` +
			`{"relativePath":"dxgi.dll","sha256":"runtime","sizeBytes":7,"ownership":"managed","role":"runtime"},` +
			`{"relativePath":"ReShade.ini","sha256":"config","sizeBytes":7,"ownership":"managed","role":"configuration"},` +
			`{"relativePath":"ReShadePreset.ini","sha256":"preset","sizeBytes":7,"ownership":"managed","role":"preset"}` +
			`],"variantProvenance":"verified"}`,
	}
	preview, err := planner.Plan(context.Background(), root, Request{
		Action:                 ActionUninstall,
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           RenderingAPID3D11,
		ProxyFilename:          "dxgi.dll",
		Architecture:           ArchitectureX64,
		BuildVariant:           BuildVariantStandard,
	}, existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Operations) != 3 {
		t.Fatalf("operations = %+v", preview.Operations)
	}
	for _, operation := range preview.Operations {
		if operation.Type != "delete" {
			t.Fatalf("expected delete operations only: %+v", operation)
		}
	}
}

func TestDirectXPlannerForeignProxyBlocksInstall(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, name := range []string{"Game.exe", "dxgi.dll"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	planner, _ := testDirectXPlanner(t)
	preview, err := planner.Plan(context.Background(), root, Request{
		Action:                 ActionInstall,
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           RenderingAPID3D11,
		ProxyFilename:          "dxgi.dll",
		Architecture:           ArchitectureX64,
		BuildVariant:           BuildVariantStandard,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Conflicts) != 1 ||
		len(preview.PathImpacts) != 1 ||
		!preview.PathImpacts[0].Blocking {
		t.Fatalf("preview = %+v", preview)
	}
}

func TestDirectXPlannerInstallSupportsOpenGL(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Game.exe"), []byte("game"), 0o644); err != nil {
		t.Fatal(err)
	}
	planner, prepared := testDirectXPlanner(t)
	preview, err := planner.Plan(context.Background(), root, Request{
		Action:                 ActionInstall,
		GameID:                 1,
		TargetRelativePath:     ".",
		ExecutableRelativePath: "Game.exe",
		RenderingAPI:           RenderingAPIOpenGL,
		ProxyFilename:          "opengl32.dll",
		Architecture:           ArchitectureX64,
		BuildVariant:           BuildVariantStandard,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Operations) == 0 ||
		preview.Operations[0].SourcePath != prepared["opengl32.dll"].Path ||
		filepath.Base(preview.Operations[0].TargetPath) != "opengl32.dll" {
		t.Fatalf("operations = %+v", preview.Operations)
	}
	if preview.DesiredTarget == nil ||
		len(preview.DesiredTarget.Manifest.Files) == 0 ||
		preview.DesiredTarget.Manifest.Files[0].RelativePath != "opengl32.dll" {
		t.Fatalf("desired target = %+v", preview.DesiredTarget)
	}
}

func testDirectXPlanner(t *testing.T) (Planner, map[string]PreparedSetupFile) {
	t.Helper()
	root := t.TempDir()
	prepared := map[string]PreparedSetupFile{}
	for _, name := range []string{"dxgi.dll", "d3d11.dll", "opengl32.dll", "ReShade.ini", "ReShadePreset.ini"} {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		hash, size, err := fileops.FileIntegrity(path)
		if err != nil {
			t.Fatal(err)
		}
		prepared[name] = PreparedSetupFile{
			RelativePath: name,
			Path:         path,
			SHA256:       hash,
			SizeBytes:    size,
		}
	}
	artifact := InstallerArtifact{
		InstallerRelease: InstallerRelease{
			Version:   "6.7.3",
			Variant:   InstallerVariantStandard,
			AssetName: "ReShade_Setup_6.7.3.exe",
			URL:       "https://reshade.me/downloads/ReShade_Setup_6.7.3.exe",
		},
		Path:      filepath.Join(root, "installer.exe"),
		SizeBytes: 100,
		SHA256:    strings.Repeat("a", 64),
	}
	planner := NewDirectXPlanner(DirectXPlannerOptions{
		ResolveInstaller: func(
			context.Context,
			InstallerVariant,
			InstallerResolveOptions,
		) (InstallerRelease, error) {
			return artifact.InstallerRelease, nil
		},
		AcquireInstaller: func(
			_ context.Context,
			release InstallerRelease,
			_ InstallerAcquireOptions,
		) (InstallerArtifact, error) {
			artifact.InstallerRelease = release
			return artifact, nil
		},
		PrepareSetup: func(
			context.Context,
			SetupRequest,
			SetupRunnerOptions,
		) (SetupRunResult, error) {
			files := make([]PreparedSetupFile, 0, len(prepared))
			for _, file := range prepared {
				files = append(files, file)
			}
			return SetupRunResult{
				Prepared: &PreparedSetup{
					Files: files,
				},
			}, nil
		},
		InspectArchitecture: func(string) (Architecture, error) {
			return ArchitectureX64, nil
		},
		ReadMetadata: func(string) (winversion.Metadata, error) {
			return winversion.Metadata{}, nil
		},
	})
	return planner, prepared
}
