package optiscaler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

type Store interface {
	GetOptiScalerTarget(context.Context, int64, string) (dbtypes.OptiScalerTarget, bool, error)
	ListOptiScalerTargets(context.Context, int64) ([]dbtypes.OptiScalerTarget, error)
	SaveOptiScalerTarget(context.Context, dbtypes.SaveOptiScalerTargetInput) (dbtypes.OptiScalerTarget, error)
	DeleteOptiScalerTarget(context.Context, int64, string) error
}

type ManagerOptions struct {
	DataDir        string
	CacheDir       string
	ReleasesURL    string
	HTTPClient     *http.Client
	Now            func() time.Time
	PreparePackage func(context.Context) (Release, Package, error)
}

type Manager struct {
	store                  Store
	dataDir                string
	cacheDir               string
	releasesURL            string
	httpClient             *http.Client
	now                    func() time.Time
	preparePackageOverride func(context.Context) (Release, Package, error)
	mu                     sync.Mutex
	packageMu              sync.Mutex
}

func NewManager(store Store, options ManagerOptions) *Manager {
	dataDir := options.DataDir
	if dataDir == "" {
		dataDir = filepath.Join(application.Path(application.PathDataHome), "fiach", "optiscaler")
	}
	cacheDir := options.CacheDir
	if cacheDir == "" {
		cacheDir = filepath.Join(application.Path(application.PathCacheHome), "fiach", "optiscaler", "releases")
	}
	if options.HTTPClient == nil {
		options.HTTPClient = http.DefaultClient
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Manager{
		store: store, dataDir: dataDir, cacheDir: cacheDir,
		releasesURL: options.ReleasesURL, httpClient: options.HTTPClient, now: options.Now,
		preparePackageOverride: options.PreparePackage,
	}
}

func (m *Manager) Discover(ctx context.Context, gameRoot string, gameID int64) (candidates []Candidate, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("discover managed game targets: %w", err)
		}
	}()
	targets, err := m.store.ListOptiScalerTargets(ctx, gameID)
	if err != nil {
		return nil, err
	}
	managed := make([]string, 0, len(targets))
	for _, target := range targets {
		managed = append(managed, target.TargetRelativePath)
	}
	return DiscoverCandidates(gameRoot, managed)
}

func (m *Manager) StableRelease(ctx context.Context) (release Release, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get OptiScaler stable release: %w", err)
		}
	}()
	return DiscoverStableRelease(ctx, m.releaseOptions())
}

func (m *Manager) Preview(ctx context.Context, gameRoot string, request Request) (preview Preview, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("preview OptiScaler action: %w", err)
		}
	}()

	request, targetPath, executablePath, err := normalizeRequest(gameRoot, request)
	if err != nil {
		return Preview{}, err
	}
	isX64, err := IsX64PE(executablePath)
	if err != nil || !isX64 {
		if err == nil {
			err = errors.New("selected executable is not x64")
		}
		return Preview{}, err
	}
	target, found, err := m.store.GetOptiScalerTarget(ctx, request.GameID, request.TargetRelativePath)
	if err != nil {
		return Preview{}, err
	}
	if request.Action == ActionInstall || request.Action == ActionAdopt {
		if found {
			return Preview{}, errors.New("target is already managed")
		}
	} else if !found {
		return Preview{}, errors.New("target is not managed")
	}

	preview = Preview{Request: request, Operations: []Operation{}, ConfigurationChanges: []string{}, Warnings: []string{}, Conflicts: []string{}, Drift: []Drift{}}
	if (!found || target.WarningAcknowledgedAt == nil || target.WarningVersion != WarningVersion) && !request.AcknowledgeWarning {
		preview.Conflicts = append(preview.Conflicts, "Online-game and anti-cheat warning acknowledgement is required.")
	}
	if request.GraphicsAPI == GraphicsAPIVulkan && request.EnableReShadeCoexistence {
		preview.Conflicts = append(preview.Conflicts, "Automated Vulkan and ReShade coexistence is not supported.")
	}

	switch request.Action {
	case ActionInstall, ActionUpdate, ActionRepair:
		release, pkg, err := m.preparePackage(ctx)
		if err != nil {
			return Preview{}, err
		}
		preview.Release = release
		if found && request.Action == ActionRepair {
			preview.Request = requestFromStoredTarget(request.Action, target, request)
			manifest, manifestErr := decodeManifest(target.ManifestJSON)
			if manifestErr != nil {
				return Preview{}, manifestErr
			}
			preview.Request.EnableReShadeCoexistence = manifest.Config.LoadReShade
		}
		operations, configChanges, conflicts, err := m.packageOperations(targetPath, preview.Request, pkg)
		if err != nil {
			return Preview{}, err
		}
		preview.Operations = operations
		preview.ConfigurationChanges = configChanges
		preview.Conflicts = append(preview.Conflicts, conflicts...)
	case ActionAdopt:
		release, pkg, err := m.preparePackage(ctx)
		if err != nil {
			return Preview{}, err
		}
		preview.Release = release
		operations, conflicts, err := adoptionInventory(targetPath, request, pkg)
		if err != nil {
			return Preview{}, err
		}
		preview.Operations = operations
		preview.Conflicts = append(preview.Conflicts, conflicts...)
		preview.Warnings = append(preview.Warnings, "Adopted files will be removed by uninstall; pre-adoption originals cannot be restored.")
	case ActionUninstall:
		preview.Release = releaseFromStoredTarget(target)
		manifest, err := decodeManifest(target.ManifestJSON)
		if err != nil {
			return Preview{}, err
		}
		preview.Operations = uninstallOperations(targetPath, manifest)
	default:
		return Preview{}, fmt.Errorf("unsupported action %q", request.Action)
	}

	if found && request.Action != ActionInstall && request.Action != ActionAdopt {
		manifest, err := decodeManifest(target.ManifestJSON)
		if err != nil {
			return Preview{}, err
		}
		preview.Drift, err = detectDrift(targetPath, manifest)
		if err != nil {
			return Preview{}, err
		}
		if len(preview.Drift) > 0 && !request.BackupAndContinue {
			preview.Conflicts = append(preview.Conflicts, "Managed files have drifted; backup-and-continue must be explicitly selected.")
		}
		status := "managed"
		if len(preview.Drift) > 0 {
			status = "drifted"
		}
		if target.Status != status {
			if err := m.saveTargetStatus(ctx, target, status); err != nil {
				return Preview{}, err
			}
		}
	}
	preview.CanApply = len(preview.Conflicts) == 0
	preview.PreviewHash, err = hashPreview(preview)
	if err != nil {
		return Preview{}, err
	}
	return preview, nil
}

func (m *Manager) Apply(ctx context.Context, gameRoot string, request Request, previewHash string) (result ApplyResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("apply OptiScaler action: %w", err)
		}
	}()
	m.mu.Lock()
	defer m.mu.Unlock()

	if recovery, err := m.RecoveryState(); err != nil {
		return ApplyResult{}, err
	} else if recovery.Required {
		return ApplyResult{}, errors.New("an OptiScaler operation requires recovery")
	}
	preview, err := m.Preview(ctx, gameRoot, request)
	if err != nil {
		return ApplyResult{}, err
	}
	if !preview.CanApply {
		return ApplyResult{}, errors.New("preview has blocking conflicts")
	}
	if previewHash == "" || !strings.EqualFold(preview.PreviewHash, previewHash) {
		return ApplyResult{}, errors.New("preview hash is stale or does not match")
	}
	return m.execute(ctx, gameRoot, preview)
}

func normalizeRequest(gameRoot string, request Request) (Request, string, string, error) {
	if request.GameID <= 0 {
		return Request{}, "", "", errors.New("game ID must be positive")
	}
	if !slices.Contains([]Action{ActionInstall, ActionAdopt, ActionUpdate, ActionRepair, ActionUninstall}, request.Action) {
		return Request{}, "", "", errors.New("action is invalid")
	}
	if request.GraphicsAPI != GraphicsAPIDirectX && request.GraphicsAPI != GraphicsAPIVulkan {
		return Request{}, "", "", errors.New("graphics API must be directx or vulkan")
	}
	validProxy := slices.ContainsFunc(SupportedProxyFilenames, func(value string) bool {
		return strings.EqualFold(value, request.ProxyFilename)
	})
	if !validProxy {
		return Request{}, "", "", errors.New("proxy filename is not supported")
	}
	for _, supported := range SupportedProxyFilenames {
		if strings.EqualFold(supported, request.ProxyFilename) {
			request.ProxyFilename = supported
			break
		}
	}
	targetPath, err := ResolveWithinRoot(gameRoot, request.TargetRelativePath)
	if err != nil {
		return Request{}, "", "", err
	}
	executablePath, err := ResolveWithinRoot(gameRoot, request.ExecutableRelativePath)
	if err != nil {
		return Request{}, "", "", err
	}
	if filepath.Dir(executablePath) != targetPath {
		return Request{}, "", "", errors.New("selected executable must be directly inside the target directory")
	}
	request.TargetRelativePath, err = RelativeToRoot(gameRoot, targetPath)
	if err != nil {
		return Request{}, "", "", err
	}
	request.ExecutableRelativePath, err = RelativeToRoot(gameRoot, executablePath)
	if err != nil {
		return Request{}, "", "", err
	}
	if request.ProcessFilter != nil {
		value := strings.TrimSpace(*request.ProcessFilter)
		if strings.ContainsAny(value, `/\`) {
			return Request{}, "", "", errors.New("process filter must be a filename, not a path")
		}
		request.ProcessFilter = &value
	}
	return request, targetPath, executablePath, nil
}

func (m *Manager) preparePackage(ctx context.Context) (Release, Package, error) {
	m.packageMu.Lock()
	defer m.packageMu.Unlock()

	if m.preparePackageOverride != nil {
		return m.preparePackageOverride(ctx)
	}
	release, err := DiscoverStableRelease(ctx, m.releaseOptions())
	if err != nil {
		return Release{}, Package{}, err
	}
	archivePath, err := EnsureReleaseArchive(ctx, release, m.releaseOptions())
	if err != nil {
		return Release{}, Package{}, err
	}
	stagingPath := filepath.Join(m.dataDir, "staging", safePathSegment(release.Tag), release.Digest)
	pkg, err := ExtractReleasePackage(ctx, archivePath, stagingPath)
	if err != nil {
		return Release{}, Package{}, err
	}
	return release, pkg, nil
}

func (m *Manager) releaseOptions() ReleaseOptions {
	return ReleaseOptions{ReleasesURL: m.releasesURL, CacheDir: m.cacheDir, HTTPClient: m.httpClient}
}

func (m *Manager) packageOperations(targetPath string, request Request, pkg Package) ([]Operation, []string, []string, error) {
	var operations []Operation
	var conflicts []string
	proxyPath := filepath.Join(targetPath, request.ProxyFilename)
	if _, err := os.Stat(proxyPath); err == nil {
		owner, inspectErr := InspectOwnership(proxyPath)
		switch {
		case inspectErr != nil || owner == OwnershipUnknown:
			conflicts = append(conflicts, fmt.Sprintf("Proxy ownership is unknown: %s", proxyPath))
		case owner == OwnershipReShade && request.EnableReShadeCoexistence:
			chainedPath := filepath.Join(targetPath, "ReShade64.dll")
			if _, err := os.Stat(chainedPath); err == nil {
				chainedOwner, chainedErr := InspectOwnership(chainedPath)
				if chainedErr != nil || chainedOwner != OwnershipReShade {
					conflicts = append(conflicts, fmt.Sprintf("Chained ReShade ownership is unknown: %s", chainedPath))
				}
			}
			operations = append(operations, Operation{Type: "move", SourcePath: proxyPath, TargetPath: chainedPath})
		case owner == OwnershipReShade:
			conflicts = append(conflicts, "A ReShade proxy already occupies the selected proxy filename; coexistence must be enabled.")
		case owner != OwnershipOptiScaler:
			conflicts = append(conflicts, fmt.Sprintf("Proxy ownership is incompatible: %s", proxyPath))
		}
	}
	for _, file := range pkg.Files {
		targetName := file.RelativePath
		if strings.EqualFold(filepath.Base(targetName), "OptiScaler.dll") {
			targetName = filepath.Join(filepath.Dir(targetName), request.ProxyFilename)
		}
		targetName = flattenPackageRoot(pkg.Files, targetName)
		target := filepath.Join(targetPath, targetName)
		operations = append(operations, Operation{Type: "copy", SourcePath: file.SourcePath, TargetPath: target, SHA256: file.SHA256, SizeBytes: file.SizeBytes})
	}
	configPath := filepath.Join(targetPath, "OptiScaler.ini")
	contents, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		contents = nil
	} else if err != nil {
		return nil, nil, nil, err
	}
	config := ManagedConfig{
		LoadReShade: request.EnableReShadeCoexistence, DXGISpoofing: request.DXGISpoofing,
		TargetProcessName: request.ProcessFilter, CheckForUpdate: false,
	}
	updated, err := UpdateManagedINI(contents, config)
	if err != nil {
		return nil, nil, nil, err
	}
	stagedConfig := filepath.Join(m.dataDir, "staging", "generated", hashBytes(updated)+".ini")
	if err := os.MkdirAll(filepath.Dir(stagedConfig), 0o755); err != nil {
		return nil, nil, nil, err
	}
	if err := os.WriteFile(stagedConfig, updated, 0o644); err != nil {
		return nil, nil, nil, err
	}
	configHash, configSize, err := fileops.FileIntegrity(stagedConfig)
	if err != nil {
		return nil, nil, nil, err
	}
	operations = upsertOperation(operations, Operation{Type: "copy", SourcePath: stagedConfig, TargetPath: configPath, SHA256: configHash, SizeBytes: configSize})
	changes := []string{
		"Plugins.LoadReshade=" + boolINI(config.LoadReShade),
		"Spoofing.Dxgi=" + boolINI(config.DXGISpoofing),
		"ProcessFilter.TargetProcessName=" + optionalINI(config.TargetProcessName),
		"Hotfix.CheckForUpdate=false",
	}
	return operations, changes, conflicts, nil
}

func flattenPackageRoot(files []PackageFile, relative string) string {
	if len(files) == 0 {
		return relative
	}
	firstParts := strings.Split(filepath.Clean(files[0].RelativePath), string(filepath.Separator))
	if len(firstParts) < 2 {
		return relative
	}
	root := firstParts[0]
	for _, file := range files[1:] {
		parts := strings.Split(filepath.Clean(file.RelativePath), string(filepath.Separator))
		if len(parts) < 2 || !strings.EqualFold(parts[0], root) {
			return relative
		}
	}
	parts := strings.Split(filepath.Clean(relative), string(filepath.Separator))
	if len(parts) > 1 && strings.EqualFold(parts[0], root) {
		return filepath.Join(parts[1:]...)
	}
	return relative
}

func adoptionInventory(targetPath string, request Request, pkg Package) ([]Operation, []string, error) {
	proxyPath := filepath.Join(targetPath, request.ProxyFilename)
	owner, err := InspectOwnership(proxyPath)
	if err != nil || owner != OwnershipOptiScaler {
		return nil, []string{"The selected proxy is not positively identified as OptiScaler."}, nil
	}
	var operations []Operation
	for _, packageFile := range pkg.Files {
		relative := flattenPackageRoot(pkg.Files, packageFile.RelativePath)
		if strings.EqualFold(filepath.Base(relative), "OptiScaler.ini") {
			continue
		}
		if strings.EqualFold(filepath.Base(relative), "OptiScaler.dll") {
			relative = filepath.Join(filepath.Dir(relative), request.ProxyFilename)
		}
		path := filepath.Join(targetPath, relative)
		hash, size, err := fileops.FileIntegrity(path)
		if err != nil {
			return nil, []string{fmt.Sprintf("Existing installation is incomplete: %s", relative)}, nil
		}
		if !strings.EqualFold(hash, packageFile.SHA256) || size != packageFile.SizeBytes {
			return nil, []string{fmt.Sprintf("Existing installation file does not match the verified release: %s", relative)}, nil
		}
		operations = append(operations, Operation{Type: "adopt", TargetPath: path, SHA256: hash, SizeBytes: size})
	}
	configPath := filepath.Join(targetPath, "OptiScaler.ini")
	hash, size, err := fileops.FileIntegrity(configPath)
	if err != nil {
		return nil, []string{"Existing installation has no readable OptiScaler.ini."}, nil
	}
	operations = upsertOperation(operations, Operation{Type: "adopt", TargetPath: configPath, SHA256: hash, SizeBytes: size})
	return operations, nil, nil
}

func upsertOperation(operations []Operation, operation Operation) []Operation {
	for index := range operations {
		if strings.EqualFold(operations[index].TargetPath, operation.TargetPath) {
			operations[index] = operation
			return operations
		}
	}
	return append(operations, operation)
}

func detectDrift(targetPath string, manifest Manifest) ([]Drift, error) {
	var drift []Drift
	for _, file := range manifest.Files {
		path := filepath.Join(targetPath, file.RelativePath)
		hash, size, err := fileops.FileIntegrity(path)
		if errors.Is(err, os.ErrNotExist) {
			drift = append(drift, Drift{RelativePath: file.RelativePath, ExpectedHash: file.SHA256, Missing: true})
			continue
		}
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(hash, file.SHA256) || size != file.SizeBytes {
			drift = append(drift, Drift{RelativePath: file.RelativePath, ExpectedHash: file.SHA256, ActualHash: hash})
		}
	}
	return drift, nil
}

func uninstallOperations(targetPath string, manifest Manifest) []Operation {
	operations := make([]Operation, 0, len(manifest.Files))
	for _, file := range manifest.Files {
		target := filepath.Join(targetPath, file.RelativePath)
		if file.BackupPath != "" {
			operations = append(operations, Operation{Type: "restore", SourcePath: file.BackupPath, TargetPath: target, SHA256: file.BackupSHA256, SizeBytes: file.BackupSize})
		} else {
			operations = append(operations, Operation{Type: "delete", TargetPath: target})
		}
	}
	return operations
}

func requestFromStoredTarget(action Action, target dbtypes.OptiScalerTarget, request Request) Request {
	request.Action = action
	request.TargetRelativePath = target.TargetRelativePath
	request.ExecutableRelativePath = target.ExecutableRelativePath
	request.GraphicsAPI = GraphicsAPI(target.GraphicsAPI)
	request.ProxyFilename = target.ProxyFilename
	request.DXGISpoofing = target.DXGISpoofing
	request.ProcessFilter = target.ProcessFilter
	return request
}

func releaseFromStoredTarget(target dbtypes.OptiScalerTarget) Release {
	return Release{Tag: target.ReleaseTag, Version: target.ReleaseVersion, AssetName: target.ReleaseAssetName, Digest: target.ReleaseDigest}
}

func decodeManifest(value string) (Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal([]byte(value), &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode OptiScaler manifest: %w", err)
	}
	if manifest.Version != ManifestVersion {
		return Manifest{}, fmt.Errorf("unsupported OptiScaler manifest version %d", manifest.Version)
	}
	return manifest, nil
}

func hashPreview(preview Preview) (string, error) {
	preview.PreviewHash = ""
	encoded, err := json.Marshal(preview)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func hashBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func (m *Manager) saveTargetStatus(ctx context.Context, target dbtypes.OptiScalerTarget, status string) error {
	_, err := m.store.SaveOptiScalerTarget(ctx, dbtypes.SaveOptiScalerTargetInput{
		GameID: target.GameID, TargetRelativePath: target.TargetRelativePath,
		ExecutableRelativePath: target.ExecutableRelativePath, GraphicsAPI: target.GraphicsAPI,
		ProxyFilename: target.ProxyFilename, DXGISpoofing: target.DXGISpoofing,
		ProcessFilter: target.ProcessFilter, ReleaseTag: target.ReleaseTag,
		ReleaseVersion: target.ReleaseVersion, ReleaseAssetName: target.ReleaseAssetName,
		ReleaseDigest: target.ReleaseDigest, ManagementOrigin: target.ManagementOrigin,
		Status: status, ManifestJSON: target.ManifestJSON, WarningVersion: target.WarningVersion,
		WarningAcknowledgedAt: target.WarningAcknowledgedAt, LastVerifiedAt: target.LastVerifiedAt,
	})
	return err
}
