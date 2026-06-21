package reshade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/filetxn"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

type Store interface {
	GetReShadeTarget(context.Context, int64, string) (dbtypes.ReShadeTarget, bool, error)
	ListReShadeTargets(context.Context, int64) ([]dbtypes.ReShadeTarget, error)
	SaveReShadeTarget(context.Context, dbtypes.SaveReShadeTargetInput) (dbtypes.ReShadeTarget, error)
	DeleteReShadeTarget(context.Context, int64, string) error
}

type Planner interface {
	Plan(context.Context, string, Request, *dbtypes.ReShadeTarget) (Preview, error)
}

type PlannerFunc func(context.Context, string, Request, *dbtypes.ReShadeTarget) (Preview, error)

func (fn PlannerFunc) Plan(ctx context.Context, gameRoot string, request Request, target *dbtypes.ReShadeTarget) (Preview, error) {
	return fn(ctx, gameRoot, request, target)
}

type ManagerOptions struct {
	DataDir           string
	Now               func() time.Time
	Planner           Planner
	VerifyApplied     func(string, Preview) error
	ExecuteOperation  func(Operation) error
	RollbackSnapshots func([]filetxn.Snapshot) error
}

type Manager struct {
	store             Store
	dataDir           string
	now               func() time.Time
	planner           Planner
	verifyApplied     func(string, Preview) error
	executeOperation  func(Operation) error
	rollbackSnapshots func([]filetxn.Snapshot) error
	mu                sync.Mutex
}

func NewManager(store Store, options ManagerOptions) *Manager {
	if options.DataDir == "" {
		options.DataDir = filepath.Join(application.Path(application.PathDataHome), "fiach", "reshade")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	productionPlanner := options.Planner == nil
	if productionPlanner {
		options.Planner = NewDirectXPlanner(DirectXPlannerOptions{})
	}
	if options.VerifyApplied == nil {
		options.VerifyApplied = func(string, Preview) error {
			return nil
		}
		if productionPlanner {
			options.VerifyApplied = verifyAppliedReShadeState
		}
	}
	if options.ExecuteOperation == nil {
		options.ExecuteOperation = func(operation Operation) error {
			return filetxn.ExecuteOperation(operation, "staged ReShade file")
		}
	}
	if options.RollbackSnapshots == nil {
		options.RollbackSnapshots = filetxn.RollbackSnapshots
	}
	return &Manager{
		store:             store,
		dataDir:           options.DataDir,
		now:               options.Now,
		planner:           options.Planner,
		verifyApplied:     options.VerifyApplied,
		executeOperation:  options.ExecuteOperation,
		rollbackSnapshots: options.RollbackSnapshots,
	}
}

func (m *Manager) ListTargets(ctx context.Context, gameRoot string, gameID int64) (targets []ManagedTarget, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list managed ReShade targets: %w", err)
		}
	}()
	rows, err := m.store.ListReShadeTargets(ctx, gameID)
	if err != nil {
		return nil, err
	}
	targets = make([]ManagedTarget, 0, len(rows))
	for _, row := range rows {
		status := ManagementStatus(row.Status)
		manifest, manifestErr := DecodeManifest(row.ManifestJSON)
		if manifestErr != nil {
			status = ManagementStatusIncompatibleManifest
		} else if status != ManagementStatusRecoveryRequired {
			targetPath, resolveErr := ResolveWithinRoot(gameRoot, row.TargetRelativePath)
			if resolveErr != nil {
				return nil, resolveErr
			}
			drift, driftErr := detectManifestDrift(targetPath, manifest)
			if driftErr != nil {
				return nil, driftErr
			}
			status = ManagementStatusManaged
			if len(drift) > 0 {
				status = ManagementStatusDrifted
			}
			if row.Status != string(status) {
				row.Status = string(status)
				if _, saveErr := m.store.SaveReShadeTarget(ctx, dbInputFromRow(row)); saveErr != nil {
					return nil, saveErr
				}
			}
		}
		targets = append(targets, managedTargetFromRow(row, status, manifest))
	}
	return targets, nil
}

func (m *Manager) ListContentCatalogue(ctx context.Context, refresh bool) (ContentCatalogue, error) {
	return ListContentCatalogue(ctx, m.dataDir, refresh, ContentCatalogueOptions{})
}

func (m *Manager) Preview(ctx context.Context, gameRoot string, request Request) (preview Preview, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("preview managed ReShade action: %w", err)
		}
	}()
	request, targetPath, err := normalizeRequest(gameRoot, request)
	if err != nil {
		return Preview{}, err
	}
	row, found, err := m.store.GetReShadeTarget(ctx, request.GameID, request.TargetRelativePath)
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

	var existing *dbtypes.ReShadeTarget
	var manifest Manifest
	if found {
		existing = &row
		manifest, err = DecodeManifest(row.ManifestJSON)
		if err != nil {
			return Preview{}, err
		}
	}
	if request.Action == ActionConfigureContent {
		if !found {
			return Preview{}, errors.New("target is not managed")
		}
		preview, err = m.planContent(ctx, gameRoot, targetPath, request, row, manifest)
	} else {
		preview, err = m.planner.Plan(ctx, gameRoot, request, existing)
	}
	if err != nil {
		return Preview{}, err
	}
	preview.Request = request
	if preview.Operations == nil {
		preview.Operations = []Operation{}
	}
	if preview.PathImpacts == nil {
		preview.PathImpacts = []PathImpact{}
	}
	if preview.Warnings == nil {
		preview.Warnings = []string{}
	}
	if preview.Conflicts == nil {
		preview.Conflicts = []string{}
	}
	if preview.Drift == nil {
		preview.Drift = []Drift{}
	}
	if preview.UserContentDrift == nil {
		preview.UserContentDrift = []UserContentDrift{}
	}
	if found && request.Action != ActionAdopt {
		preview.Drift, err = detectManifestDrift(targetPath, manifest)
		if err != nil {
			return Preview{}, err
		}
		if len(preview.Drift) > 0 && !request.BackupAndContinue {
			preview.Conflicts = append(preview.Conflicts,
				"Managed ReShade files have drifted; backup-and-continue must be explicitly selected.")
		}
		preview.UserContentDrift, err = detectUserContentDrift(gameRoot, manifest)
		if err != nil {
			return Preview{}, err
		}
	}
	if err := m.annotateOperationBackups(targetPath, request, manifest, found, preview.Operations); err != nil {
		return Preview{}, err
	}
	for _, operation := range preview.Operations {
		if strings.TrimSpace(operation.BackupPath) == "" {
			continue
		}
		preview.PathImpacts = append(preview.PathImpacts, PathImpact{
			Path:             operation.BackupPath,
			Role:             PathRoleBackup,
			Action:           "create",
			Ownership:        OwnershipManaged,
			Exists:           pathExists(operation.BackupPath),
			PreservationOnly: true,
		})
	}
	if err := validatePlannedMutation(targetPath, preview, manifest, found); err != nil {
		return Preview{}, err
	}
	preview.CanApply = len(preview.Conflicts) == 0
	preview.PreviewHash, err = hashPreview(preview)
	if err != nil {
		return Preview{}, err
	}
	return preview, nil
}

func (m *Manager) annotateOperationBackups(
	targetPath string,
	request Request,
	existing Manifest,
	found bool,
	operations []Operation,
) error {
	previousBackups := map[string]string{}
	previouslyOwned := map[string]bool{}
	if found {
		for _, file := range existing.Files {
			key := strings.ToLower(filepath.Clean(file.RelativePath))
			if file.Ownership == OwnershipManaged || file.Ownership == OwnershipAdopted {
				previouslyOwned[key] = true
			}
			if file.BackupPath != nil {
				previousBackups[key] = *file.BackupPath
			}
		}
	}
	targetKey := hashBytes([]byte(strings.ToLower(request.TargetRelativePath)))[:16]
	for index := range operations {
		operation := &operations[index]
		if operation.Type == "adopt" || operation.Type == "delete" || operation.Type == "restore" {
			continue
		}
		info, err := os.Stat(operation.TargetPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("mutation target %q is not a regular file", operation.TargetPath)
		}
		relative, err := filepath.Rel(targetPath, operation.TargetPath)
		if err != nil {
			return err
		}
		key := strings.ToLower(filepath.Clean(relative))
		if previous := previousBackups[key]; previous != "" {
			operation.BackupPath = previous
			continue
		}
		if previouslyOwned[key] {
			continue
		}
		hash, _, err := fileops.FileIntegrity(operation.TargetPath)
		if err != nil {
			return err
		}
		operation.BackupPath = filepath.Join(
			m.dataDir, "backups", fmt.Sprintf("%d", request.GameID), targetKey,
			fmt.Sprintf("%s-%s.bak", hash[:16], filetxn.SafePathSegment(filepath.Base(relative))),
		)
	}
	return nil
}

func (m *Manager) Apply(ctx context.Context, gameRoot string, request Request, previewHash string) (result ApplyResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("apply managed ReShade action: %w", err)
		}
	}()
	m.mu.Lock()
	defer m.mu.Unlock()

	recovery, err := m.RecoveryState()
	if err != nil {
		return ApplyResult{}, err
	}
	if recovery.Required {
		return ApplyResult{}, errors.New("a ReShade operation requires recovery")
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

func normalizeRequest(gameRoot string, request Request) (Request, string, error) {
	if request.GameID <= 0 {
		return Request{}, "", errors.New("game ID must be positive")
	}
	if !slices.Contains([]Action{
		ActionInstall,
		ActionAdopt,
		ActionUpdate,
		ActionRepair,
		ActionUninstall,
		ActionConfigureContent,
	}, request.Action) {
		return Request{}, "", errors.New("action is invalid")
	}
	if !slices.Contains([]RenderingAPI{RenderingAPID3D9, RenderingAPID3D10, RenderingAPID3D11, RenderingAPID3D12}, request.RenderingAPI) {
		return Request{}, "", errors.New("rendering API is invalid")
	}
	if !slices.Contains([]Architecture{ArchitectureX86, ArchitectureX64}, request.Architecture) {
		return Request{}, "", errors.New("architecture is invalid")
	}
	if !slices.Contains([]BuildVariant{BuildVariantStandard, BuildVariantAddon}, request.BuildVariant) {
		return Request{}, "", errors.New("build variant is invalid")
	}
	if request.BuildVariant == BuildVariantAddon &&
		request.Action != ActionUninstall &&
		(!request.SinglePlayerAcknowledged || !request.AntiCheatRiskAcknowledged) {
		return Request{}, "", errors.New(
			"full add-on build requires separate single-player and anti-cheat risk acknowledgements")
	}
	if strings.TrimSpace(request.ProxyFilename) == "" {
		return Request{}, "", errors.New("proxy filename is required")
	}
	if !proxyAllowedForAPI(request.RenderingAPI, request.ProxyFilename) {
		return Request{}, "", fmt.Errorf(
			"proxy filename %q is not supported for rendering API %q",
			request.ProxyFilename,
			request.RenderingAPI,
		)
	}
	targetPath, err := ResolveWithinRoot(gameRoot, request.TargetRelativePath)
	if err != nil {
		return Request{}, "", err
	}
	executablePath, err := ResolveWithinRoot(gameRoot, request.ExecutableRelativePath)
	if err != nil {
		return Request{}, "", err
	}
	if filepath.Dir(executablePath) != targetPath {
		return Request{}, "", errors.New("selected executable must be directly inside the target directory")
	}
	return request, targetPath, nil
}

func validatePlannedMutation(targetPath string, preview Preview, existing Manifest, found bool) error {
	if err := filetxn.ValidateOperations(preview.Operations, targetPath); err != nil {
		return err
	}
	manifest := existing
	ownership := map[string]Ownership{}
	for _, file := range existing.Files {
		ownership[strings.ToLower(filepath.Clean(file.RelativePath))] = file.Ownership
	}
	if preview.DesiredTarget != nil {
		manifest = preview.DesiredTarget.Manifest
		encoded, err := json.Marshal(manifest)
		if err != nil {
			return err
		}
		if _, err := DecodeManifest(string(encoded)); err != nil {
			return err
		}
	} else if !found && len(preview.Operations) > 0 {
		return errors.New("planned mutation has no desired target manifest")
	}
	for _, file := range manifest.Files {
		ownership[strings.ToLower(filepath.Clean(file.RelativePath))] = file.Ownership
	}
	for _, operation := range preview.Operations {
		relative, err := filepath.Rel(targetPath, operation.TargetPath)
		if err != nil {
			return err
		}
		owner := ownership[strings.ToLower(filepath.Clean(relative))]
		if owner != OwnershipManaged && owner != OwnershipAdopted &&
			!(operation.Type == "copy" && owner == OwnershipUser) {
			return fmt.Errorf("operation target %q is not manifest-owned", relative)
		}
	}
	return nil
}

func proxyAllowedForAPI(renderingAPI RenderingAPI, proxyFilename string) bool {
	proxyFilename = strings.ToLower(strings.TrimSpace(proxyFilename))
	switch renderingAPI {
	case RenderingAPID3D9:
		return proxyFilename == "d3d9.dll"
	case RenderingAPID3D10:
		return slices.Contains([]string{"d3d10.dll", "d3d10core.dll", "dxgi.dll"}, proxyFilename)
	case RenderingAPID3D11:
		return slices.Contains([]string{"d3d11.dll", "dxgi.dll"}, proxyFilename)
	case RenderingAPID3D12:
		return slices.Contains([]string{"d3d12.dll", "dxgi.dll"}, proxyFilename)
	default:
		return false
	}
}

func hashPreview(preview Preview) (string, error) {
	preview.PreviewHash = ""
	contents, err := json.Marshal(preview)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(contents)
	return hex.EncodeToString(sum[:]), nil
}

func hashBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func managedTargetFromRow(
	row dbtypes.ReShadeTarget,
	status ManagementStatus,
	manifest Manifest,
) ManagedTarget {
	return ManagedTarget{
		ID:                     row.ID,
		GameID:                 row.GameID,
		TargetRelativePath:     row.TargetRelativePath,
		ExecutableRelativePath: row.ExecutableRelativePath,
		RenderingAPI:           RenderingAPI(row.RenderingAPI),
		ProxyFilename:          row.ProxyFilename,
		Architecture:           Architecture(row.Architecture),
		BuildVariant:           BuildVariant(row.BuildVariant),
		VariantProvenance:      manifest.VariantProvenance,
		RuntimeVersion:         row.RuntimeVersion,
		Provenance: InstallerProvenance{
			Tag:       row.InstallerTag,
			AssetName: row.InstallerAssetName,
			URL:       row.InstallerURL,
			Digest:    row.InstallerDigest,
			Size:      row.InstallerSize,
		},
		ManagementOrigin: row.ManagementOrigin,
		Status:           status,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		LastVerifiedAt:   row.LastVerifiedAt,
	}
}

func copyArchiveFile(source string, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
		SourcePath: source, TargetPath: target, Mode: 0o644, OpenLabel: "ReShade archive file",
	})
}
