package reshade

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/filetxn"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

type memoryReShadeStore struct {
	targets map[string]dbtypes.ReShadeTarget
}

func newMemoryReShadeStore() *memoryReShadeStore {
	return &memoryReShadeStore{targets: map[string]dbtypes.ReShadeTarget{}}
}

func (s *memoryReShadeStore) key(gameID int64, path string) string {
	return strings.ToLower(strings.Join([]string{strconv.FormatInt(gameID, 10), filepath.Clean(path)}, ":"))
}

func (s *memoryReShadeStore) GetReShadeTarget(_ context.Context, gameID int64, path string) (dbtypes.ReShadeTarget, bool, error) {
	target, found := s.targets[s.key(gameID, path)]
	return target, found, nil
}

func (s *memoryReShadeStore) ListReShadeTargets(_ context.Context, gameID int64) ([]dbtypes.ReShadeTarget, error) {
	var result []dbtypes.ReShadeTarget
	for _, target := range s.targets {
		if target.GameID == gameID {
			result = append(result, target)
		}
	}
	return result, nil
}

func (s *memoryReShadeStore) SaveReShadeTarget(_ context.Context, input dbtypes.SaveReShadeTargetInput) (dbtypes.ReShadeTarget, error) {
	target := dbtypes.ReShadeTarget{
		ID: 1, GameID: input.GameID, TargetRelativePath: input.TargetRelativePath,
		ExecutableRelativePath: input.ExecutableRelativePath, RenderingAPI: input.RenderingAPI,
		ProxyFilename: input.ProxyFilename, Architecture: input.Architecture,
		BuildVariant: input.BuildVariant, RuntimeVersion: input.RuntimeVersion,
		InstallerTag: input.InstallerTag, InstallerAssetName: input.InstallerAssetName,
		InstallerURL: input.InstallerURL, InstallerDigest: input.InstallerDigest,
		InstallerSize: input.InstallerSize, ManagementOrigin: input.ManagementOrigin,
		Status: input.Status, ManifestJSON: input.ManifestJSON, LastVerifiedAt: input.LastVerifiedAt,
	}
	s.targets[s.key(input.GameID, input.TargetRelativePath)] = target
	return target, nil
}

func (s *memoryReShadeStore) DeleteReShadeTarget(_ context.Context, gameID int64, path string) error {
	delete(s.targets, s.key(gameID, path))
	return nil
}

func TestManagerProductionPlannerReturnsBlockingPreview(t *testing.T) {
	t.Parallel()
	root, request := newReShadeRequest(t)
	manager := NewManager(newMemoryReShadeStore(), ManagerOptions{DataDir: t.TempDir()})
	preview, err := manager.Preview(context.Background(), root, request)
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if preview.CanApply || len(preview.Conflicts) != 1 || preview.PreviewHash == "" {
		t.Fatalf("Preview() = %+v", preview)
	}
	if _, err := manager.Apply(context.Background(), root, request, preview.PreviewHash); err == nil {
		t.Fatal("Apply() error = nil")
	}
}

func TestManagerAppliesInjectedPlanAndRejectsStaleHash(t *testing.T) {
	t.Parallel()
	root, request := newReShadeRequest(t)
	source := filepath.Join(t.TempDir(), "ReShade64.dll")
	if err := os.WriteFile(source, []byte("runtime"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, size, err := fileops.FileIntegrity(source)
	if err != nil {
		t.Fatal(err)
	}
	planner := PlannerFunc(func(_ context.Context, gameRoot string, request Request, _ *dbtypes.ReShadeTarget) (Preview, error) {
		target, err := ResolveWithinRoot(gameRoot, filepath.Join(request.TargetRelativePath, request.ProxyFilename))
		if err != nil {
			return Preview{}, err
		}
		return Preview{
			Operations: []Operation{{Type: "copy", SourcePath: source, TargetPath: target, SHA256: hash, SizeBytes: size}},
			DesiredTarget: &TargetState{
				RuntimeVersion: "6.5.1", ManagementOrigin: "installed",
				Manifest: Manifest{Version: ManifestVersion, HasPreAdoptionRollbackData: true, Files: []ManagedFile{{
					RelativePath: request.ProxyFilename, SHA256: hash, SizeBytes: size, Ownership: OwnershipManaged,
				}}},
			},
		}, nil
	})
	store := newMemoryReShadeStore()
	manager := NewManager(store, ManagerOptions{DataDir: t.TempDir(), Planner: planner})
	preview, err := manager.Preview(context.Background(), root, request)
	if err != nil || !preview.CanApply {
		t.Fatalf("Preview() = %+v, %v", preview, err)
	}
	if _, err := manager.Apply(context.Background(), root, request, strings.Repeat("0", 64)); err == nil {
		t.Fatal("Apply(stale hash) error = nil")
	}
	result, err := manager.Apply(context.Background(), root, request, preview.PreviewHash)
	if err != nil || !result.Success {
		t.Fatalf("Apply() = %+v, %v", result, err)
	}
	if _, found, _ := store.GetReShadeTarget(context.Background(), request.GameID, "."); !found {
		t.Fatal("managed target was not committed")
	}
}

func TestManagerPersistsPreexistingFileBackupInManifest(t *testing.T) {
	t.Parallel()
	root, request := newReShadeRequest(t)
	targetPath := filepath.Join(root, request.ProxyFilename)
	if err := os.WriteFile(targetPath, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "ReShade64.dll")
	if err := os.WriteFile(source, []byte("replacement"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, size, err := fileops.FileIntegrity(source)
	if err != nil {
		t.Fatal(err)
	}
	store := newMemoryReShadeStore()
	manager := NewManager(store, ManagerOptions{
		DataDir: t.TempDir(),
		Planner: testPlanner(source, hash, size),
	})
	preview, err := manager.Preview(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Operations[0].BackupPath == "" {
		t.Fatal("preview operation has no persistent backup path")
	}
	if _, err := manager.Apply(context.Background(), root, request, preview.PreviewHash); err != nil {
		t.Fatal(err)
	}
	row, found, err := store.GetReShadeTarget(context.Background(), request.GameID, ".")
	if err != nil || !found {
		t.Fatalf("GetReShadeTarget() = %+v, %v, %v", row, found, err)
	}
	manifest, err := DecodeManifest(row.ManifestJSON)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Files) != 1 || manifest.Files[0].BackupPath == nil ||
		manifest.Files[0].BackupSHA256 == nil || manifest.Files[0].BackupSize == nil {
		t.Fatalf("manifest backup metadata = %+v", manifest.Files)
	}
	contents, err := os.ReadFile(*manifest.Files[0].BackupPath)
	if err != nil || string(contents) != "original" {
		t.Fatalf("backup contents = %q, %v", contents, err)
	}
}

func TestManagerRollbackFailurePersistsRecovery(t *testing.T) {
	t.Parallel()
	root, request := newReShadeRequest(t)
	source := filepath.Join(t.TempDir(), "source.dll")
	if err := os.WriteFile(source, []byte("runtime"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, size, _ := fileops.FileIntegrity(source)
	planner := testPlanner(source, hash, size)
	manager := NewManager(newMemoryReShadeStore(), ManagerOptions{
		DataDir: t.TempDir(), Planner: planner,
		ExecuteOperation:  func(Operation) error { return errors.New("injected operation failure") },
		RollbackSnapshots: func([]filetxn.Snapshot) error { return errors.New("injected rollback failure") },
	})
	preview, err := manager.Preview(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Apply(context.Background(), root, request, preview.PreviewHash); err == nil {
		t.Fatal("Apply() error = nil")
	}
	recovery, err := manager.RecoveryState()
	if err != nil || !recovery.Required {
		t.Fatalf("RecoveryState() = %+v, %v", recovery, err)
	}
}

func TestManagerDriftBlocksUpdateWithoutBackupAndContinue(t *testing.T) {
	t.Parallel()
	root, request := newReShadeRequest(t)
	source := filepath.Join(t.TempDir(), "source.dll")
	if err := os.WriteFile(source, []byte("runtime"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, size, _ := fileops.FileIntegrity(source)
	store := newMemoryReShadeStore()
	manager := NewManager(store, ManagerOptions{DataDir: t.TempDir(), Planner: testPlanner(source, hash, size)})
	installPreview, err := manager.Preview(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Apply(context.Background(), root, request, installPreview.PreviewHash); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, request.ProxyFilename), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	request.Action = ActionUpdate
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
}

func TestManagerListsUnsupportedManifestAsIncompatible(t *testing.T) {
	t.Parallel()
	root, _ := newReShadeRequest(t)
	store := newMemoryReShadeStore()
	store.targets[store.key(1, ".")] = dbtypes.ReShadeTarget{
		ID: 1, GameID: 1, TargetRelativePath: ".", ExecutableRelativePath: "Game.exe",
		RenderingAPI: "d3d11", ProxyFilename: "dxgi.dll", Architecture: "x64",
		BuildVariant: "standard", RuntimeVersion: "6", ManagementOrigin: "installed",
		Status: "managed", ManifestJSON: `{"version":2,"files":[]}`,
	}
	manager := NewManager(store, ManagerOptions{DataDir: t.TempDir()})
	targets, err := manager.ListTargets(context.Background(), root, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].Status != ManagementStatusIncompatibleManifest {
		t.Fatalf("ListTargets() = %+v", targets)
	}
}

func TestDecodeManifestRejectsUnknownVersionAndOwnership(t *testing.T) {
	t.Parallel()
	for _, manifest := range []Manifest{
		{Version: ManifestVersion + 1},
		{Version: ManifestVersion, Files: []ManagedFile{{RelativePath: "dxgi.dll", SHA256: "hash", Ownership: "other"}}},
	} {
		contents, _ := json.Marshal(manifest)
		if _, err := DecodeManifest(string(contents)); err == nil {
			t.Fatalf("DecodeManifest(%s) error = nil", contents)
		}
	}
}

func newReShadeRequest(t *testing.T) (string, Request) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Game.exe"), []byte("exe"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, Request{
		Action: ActionInstall, GameID: 1, TargetRelativePath: ".",
		ExecutableRelativePath: "Game.exe", RenderingAPI: RenderingAPID3D11,
		ProxyFilename: "dxgi.dll", Architecture: ArchitectureX64,
		BuildVariant: BuildVariantStandard,
	}
}

func testPlanner(source string, hash string, size int64) Planner {
	return PlannerFunc(func(_ context.Context, gameRoot string, request Request, _ *dbtypes.ReShadeTarget) (Preview, error) {
		target := filepath.Join(gameRoot, request.ProxyFilename)
		return Preview{
			Operations: []Operation{{Type: "copy", SourcePath: source, TargetPath: target, SHA256: hash, SizeBytes: size}},
			DesiredTarget: &TargetState{
				RuntimeVersion: "6", ManagementOrigin: "installed",
				Manifest: Manifest{Version: ManifestVersion, Files: []ManagedFile{{
					RelativePath: request.ProxyFilename, SHA256: hash, SizeBytes: size, Ownership: OwnershipManaged,
				}}},
			},
		}, nil
	})
}
