package optiscaler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/phergul/fiach/internal/fileops"
)

const reShadeSessionVersion = 1

type reShadeSessionFile struct {
	Path      string    `json:"path"`
	Existed   bool      `json:"existed"`
	Ownership Ownership `json:"ownership"`
	SHA256    string    `json:"sha256,omitempty"`
	SizeBytes int64     `json:"sizeBytes,omitempty"`
	Snapshot  string    `json:"snapshot,omitempty"`
}

type reShadeSessionDocument struct {
	Version int                 `json:"version"`
	State   ReShadeSessionState `json:"state"`
	Primary reShadeSessionFile  `json:"primary"`
	Chained reShadeSessionFile  `json:"chained"`
}

func (m *Manager) StartReShadeSession(
	ctx context.Context,
	gameRoot string,
	request ReShadeSessionRequest,
) (state ReShadeSessionState, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("start coordinated ReShade session: %w", err)
		}
	}()

	m.mu.Lock()
	defer m.mu.Unlock()

	if request.InstallerVariant != ReShadeInstallerVariantStandard &&
		request.InstallerVariant != ReShadeInstallerVariantAddon {
		return ReShadeSessionState{}, errors.New("installer variant is invalid")
	}
	if existing, readErr := m.readReShadeSession(); readErr != nil {
		return ReShadeSessionState{}, readErr
	} else if existing != nil {
		return ReShadeSessionState{}, errors.New("another coordinated ReShade session is already pending")
	}
	target, found, err := m.store.GetOptiScalerTarget(ctx, request.GameID, request.TargetRelativePath)
	if err != nil {
		return ReShadeSessionState{}, err
	}
	if !found {
		return ReShadeSessionState{}, errors.New("managed OptiScaler target was not found")
	}
	if target.GraphicsAPI != string(GraphicsAPIDirectX) {
		return ReShadeSessionState{}, errors.New("automated Vulkan and ReShade coexistence is not supported")
	}
	targetPath, err := ResolveWithinRoot(gameRoot, target.TargetRelativePath)
	if err != nil {
		return ReShadeSessionState{}, err
	}
	primaryPath := filepath.Join(targetPath, target.ProxyFilename)
	chainedPath := filepath.Join(targetPath, "ReShade64.dll")
	id := fmt.Sprintf("%d-%s", m.now().UnixNano(), hashBytes([]byte(strings.ToLower(targetPath)))[:12])
	root := filepath.Join(m.dataDir, "reshade-sessions", id)
	primary, err := m.captureReShadeSessionFile(primaryPath, filepath.Join(root, "primary.bak"))
	if err != nil {
		return ReShadeSessionState{}, err
	}
	if primary.Ownership != OwnershipOptiScaler {
		return ReShadeSessionState{}, fmt.Errorf("primary proxy is not positively identified as OptiScaler: %s", primaryPath)
	}
	chained, err := m.captureReShadeSessionFile(chainedPath, filepath.Join(root, "chained.bak"))
	if err != nil {
		return ReShadeSessionState{}, err
	}
	if chained.Existed && chained.Ownership != OwnershipReShade {
		return ReShadeSessionState{}, fmt.Errorf("chained runtime ownership is unknown: %s", chainedPath)
	}
	state = ReShadeSessionState{
		ID:                     id,
		GameID:                 request.GameID,
		TargetRelativePath:     target.TargetRelativePath,
		ExecutableRelativePath: target.ExecutableRelativePath,
		ProxyFilename:          target.ProxyFilename,
		ChainedFilename:        "ReShade64.dll",
		InstallerVariant:       request.InstallerVariant,
		Phase:                  ReShadeSessionPhaseAwaitingCompletion,
		StartedAt:              m.now(),
	}
	document := reShadeSessionDocument{
		Version: reShadeSessionVersion,
		State:   state,
		Primary: primary,
		Chained: chained,
	}
	if err := m.writeReShadeSession(document); err != nil {
		_ = os.RemoveAll(root)
		return ReShadeSessionState{}, err
	}
	return state, nil
}

func (m *Manager) GetReShadeSession() (state *ReShadeSessionState, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get coordinated ReShade session: %w", err)
		}
	}()
	document, err := m.readReShadeSession()
	if err != nil || document == nil {
		return nil, err
	}
	result := document.State
	return &result, nil
}

func (m *Manager) RescanReShadeSession(
	ctx context.Context,
	gameRoot string,
) (result ReShadeSessionResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("rescan coordinated ReShade session: %w", err)
		}
	}()

	m.mu.Lock()
	defer m.mu.Unlock()

	document, err := m.readReShadeSession()
	if err != nil {
		return ReShadeSessionResult{}, err
	}
	if document == nil {
		return ReShadeSessionResult{}, errors.New("no coordinated ReShade session is pending")
	}
	targetPath, err := ResolveWithinRoot(gameRoot, document.State.TargetRelativePath)
	if err != nil {
		return ReShadeSessionResult{}, err
	}
	primaryPath := filepath.Join(targetPath, document.State.ProxyFilename)
	chainedPath := filepath.Join(targetPath, document.State.ChainedFilename)
	primaryOwner, primaryExists, err := m.inspectReShadeSessionPath(primaryPath)
	if err != nil {
		return ReShadeSessionResult{}, err
	}
	chainedOwner, chainedExists, err := m.inspectReShadeSessionPath(chainedPath)
	if err != nil {
		return ReShadeSessionResult{}, err
	}

	switch {
	case primaryExists && primaryOwner == OwnershipOptiScaler &&
		(!chainedExists || chainedOwner == OwnershipReShade):
		if err := m.removeReShadeSession(*document, false); err != nil {
			return ReShadeSessionResult{}, err
		}
		return ReShadeSessionResult{
			Outcome: ReShadeSessionOutcomeHealthy,
			Message: "ReShade and OptiScaler chain state is healthy.",
		}, nil
	case primaryExists && primaryOwner == OwnershipReShade &&
		(!chainedExists || chainedOwner == OwnershipReShade):
		preview, err := m.previewReShadeRepair(ctx, gameRoot, *document, false)
		if err != nil {
			return ReShadeSessionResult{}, err
		}
		document.State.Phase = ReShadeSessionPhaseRepairReady
		document.State.ConflictingPath = ""
		document.State.Preview = &preview
		if err := m.writeReShadeSession(*document); err != nil {
			return ReShadeSessionResult{}, err
		}
		state := document.State
		return ReShadeSessionResult{
			Outcome: ReShadeSessionOutcomeRepairRequired,
			Session: &state,
			Message: "ReShade replaced the OptiScaler proxy. Review the repair preview.",
		}, nil
	default:
		conflict := primaryPath
		if primaryExists && primaryOwner == OwnershipOptiScaler && chainedExists && chainedOwner == OwnershipUnknown {
			conflict = chainedPath
		}
		document.State.Phase = ReShadeSessionPhaseConflict
		document.State.ConflictingPath = conflict
		document.State.Preview = nil
		if err := m.writeReShadeSession(*document); err != nil {
			return ReShadeSessionResult{}, err
		}
		state := document.State
		return ReShadeSessionResult{
			Outcome: ReShadeSessionOutcomeConflict,
			Session: &state,
			Message: "DLL ownership could not be determined safely.",
		}, nil
	}
}

func (m *Manager) PreviewReShadeRepair(
	ctx context.Context,
	gameRoot string,
	backupAndContinue bool,
) (preview Preview, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("preview coordinated ReShade repair: %w", err)
		}
	}()
	m.mu.Lock()
	defer m.mu.Unlock()
	document, err := m.readReShadeSession()
	if err != nil {
		return Preview{}, err
	}
	if document == nil || document.State.Phase != ReShadeSessionPhaseRepairReady {
		return Preview{}, errors.New("coordinated ReShade repair is not ready")
	}
	preview, err = m.previewReShadeRepair(ctx, gameRoot, *document, backupAndContinue)
	if err != nil {
		return Preview{}, err
	}
	document.State.Preview = &preview
	if err := m.writeReShadeSession(*document); err != nil {
		return Preview{}, err
	}
	return preview, nil
}

func (m *Manager) CancelReShadeSession(
	ctx context.Context,
	gameRoot string,
) (result ReShadeSessionResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cancel coordinated ReShade session: %w", err)
		}
	}()
	result, err = m.RescanReShadeSession(ctx, gameRoot)
	if err != nil {
		return ReShadeSessionResult{}, err
	}
	document, err := m.readReShadeSession()
	if err != nil {
		return ReShadeSessionResult{}, err
	}
	if document != nil {
		archive := result.Outcome == ReShadeSessionOutcomeConflict ||
			result.Outcome == ReShadeSessionOutcomeRepairRequired
		if err := m.removeReShadeSession(*document, archive); err != nil {
			return ReShadeSessionResult{}, err
		}
	}
	result.Outcome = ReShadeSessionOutcomeCancelled
	result.Session = nil
	result.Message = "Coordinated ReShade session cancelled without changing game files."
	return result, nil
}

func (m *Manager) DiscardReShadeSession() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("discard coordinated ReShade session: %w", err)
		}
	}()
	m.mu.Lock()
	defer m.mu.Unlock()
	document, err := m.readReShadeSession()
	if err != nil || document == nil {
		return err
	}
	return m.removeReShadeSession(*document, false)
}

func (m *Manager) ApplyReShadeRepair(
	ctx context.Context,
	gameRoot string,
	previewHash string,
) (result ApplyResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("apply coordinated ReShade repair: %w", err)
		}
	}()
	m.mu.Lock()
	defer m.mu.Unlock()
	if recovery, err := m.RecoveryState(); err != nil {
		return ApplyResult{}, err
	} else if recovery.Required {
		return ApplyResult{}, errors.New("an OptiScaler operation requires recovery")
	}
	document, err := m.readReShadeSession()
	if err != nil {
		return ApplyResult{}, err
	}
	if document == nil || document.State.Phase != ReShadeSessionPhaseRepairReady {
		return ApplyResult{}, errors.New("coordinated ReShade repair is not ready")
	}
	backupAndContinue := document.State.Preview != nil &&
		document.State.Preview.Request.BackupAndContinue
	preview, err := m.previewReShadeRepair(ctx, gameRoot, *document, backupAndContinue)
	if err != nil {
		return ApplyResult{}, err
	}
	if !preview.CanApply {
		return ApplyResult{}, errors.New("preview has blocking conflicts")
	}
	if previewHash == "" || !strings.EqualFold(previewHash, preview.PreviewHash) {
		return ApplyResult{}, errors.New("preview hash is stale or does not match")
	}
	result, err = m.execute(ctx, gameRoot, preview)
	if err != nil {
		return result, err
	}
	if err := m.removeReShadeSession(*document, false); err != nil {
		return ApplyResult{}, err
	}
	return result, nil
}

func (m *Manager) previewReShadeRepair(
	ctx context.Context,
	gameRoot string,
	document reShadeSessionDocument,
	backupAndContinue bool,
) (Preview, error) {
	target, found, err := m.store.GetOptiScalerTarget(ctx, document.State.GameID, document.State.TargetRelativePath)
	if err != nil {
		return Preview{}, err
	}
	if !found {
		return Preview{}, errors.New("managed OptiScaler target was not found")
	}
	targetPath, err := ResolveWithinRoot(gameRoot, target.TargetRelativePath)
	if err != nil {
		return Preview{}, err
	}
	manifest, err := decodeManifest(target.ManifestJSON)
	if err != nil {
		return Preview{}, err
	}
	request := Request{
		Action:                   ActionReShadeRepair,
		GameID:                   target.GameID,
		TargetRelativePath:       target.TargetRelativePath,
		ExecutableRelativePath:   target.ExecutableRelativePath,
		GraphicsAPI:              GraphicsAPIDirectX,
		ProxyFilename:            target.ProxyFilename,
		DXGISpoofing:             target.DXGISpoofing,
		ProcessFilter:            target.ProcessFilter,
		BackupAndContinue:        backupAndContinue,
		EnableReShadeCoexistence: true,
	}
	primaryPath := filepath.Join(targetPath, target.ProxyFilename)
	chainedPath := filepath.Join(targetPath, "ReShade64.dll")
	operations := []Operation{{
		Type:       "move",
		SourcePath: primaryPath,
		TargetPath: chainedPath,
	}}
	primaryHash, primarySize, err := fileops.FileIntegrity(document.Primary.Snapshot)
	if err != nil {
		return Preview{}, err
	}
	operations = append(operations, Operation{
		Type:       "copy",
		SourcePath: document.Primary.Snapshot,
		TargetPath: primaryPath,
		SHA256:     primaryHash,
		SizeBytes:  primarySize,
	})
	configPath := filepath.Join(targetPath, "OptiScaler.ini")
	contents, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		contents = nil
	} else if err != nil {
		return Preview{}, err
	}
	config := manifest.Config
	config.LoadReShade = true
	updated, err := UpdateManagedINI(contents, config)
	if err != nil {
		return Preview{}, err
	}
	stagedConfig := filepath.Join(m.dataDir, "staging", "generated", hashBytes(updated)+".ini")
	if err := os.MkdirAll(filepath.Dir(stagedConfig), 0o755); err != nil {
		return Preview{}, err
	}
	if err := os.WriteFile(stagedConfig, updated, 0o644); err != nil {
		return Preview{}, err
	}
	configHash, configSize, err := fileops.FileIntegrity(stagedConfig)
	if err != nil {
		return Preview{}, err
	}
	operations = append(operations, Operation{
		Type:       "copy",
		SourcePath: stagedConfig,
		TargetPath: configPath,
		SHA256:     configHash,
		SizeBytes:  configSize,
	})
	if err := m.annotateOperationBackups(targetPath, request, true, target, operations); err != nil {
		return Preview{}, err
	}
	drift, err := detectDrift(targetPath, manifest)
	if err != nil {
		return Preview{}, err
	}
	drift = slices.DeleteFunc(drift, func(item Drift) bool {
		name := filepath.Base(item.RelativePath)
		return strings.EqualFold(name, target.ProxyFilename) ||
			strings.EqualFold(name, "ReShade64.dll")
	})
	conflicts := []string{}
	if len(drift) > 0 && !backupAndContinue {
		conflicts = append(conflicts,
			"Managed files have drifted; backup-and-continue must be explicitly selected.")
	}
	preview := Preview{
		Request:              request,
		Release:              releaseFromStoredTarget(target),
		Operations:           operations,
		ConfigurationChanges: []string{"Plugins.LoadReshade=true"},
		Warnings:             []string{"A transaction snapshot will be created before applying changes."},
		Conflicts:            conflicts,
		Drift:                drift,
		CanApply:             len(conflicts) == 0,
	}
	preview.PreviewHash, err = hashPreview(preview)
	if err != nil {
		return Preview{}, err
	}
	return preview, nil
}

func (m *Manager) captureReShadeSessionFile(path string, snapshot string) (reShadeSessionFile, error) {
	result := reShadeSessionFile{Path: path}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	if !info.Mode().IsRegular() {
		return result, fmt.Errorf("session path is not a regular file: %s", path)
	}
	result.Existed = true
	result.Ownership, err = m.inspectOwnership(path)
	if err != nil {
		result.Ownership = OwnershipUnknown
	}
	result.SHA256, result.SizeBytes, err = fileops.FileIntegrity(path)
	if err != nil {
		return result, err
	}
	if err := os.MkdirAll(filepath.Dir(snapshot), 0o755); err != nil {
		return result, err
	}
	if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
		SourcePath: path,
		TargetPath: snapshot,
		Mode:       0o644,
		Replace:    true,
		OpenLabel:  "coordinated ReShade session file",
	}); err != nil {
		return result, err
	}
	result.Snapshot = snapshot
	return result, nil
}

func (m *Manager) inspectReShadeSessionPath(path string) (Ownership, bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return OwnershipUnknown, false, nil
	}
	if err != nil {
		return OwnershipUnknown, false, err
	}
	if !info.Mode().IsRegular() {
		return OwnershipUnknown, true, nil
	}
	ownership, err := m.inspectOwnership(path)
	if err != nil {
		return OwnershipUnknown, true, nil
	}
	return ownership, true, nil
}

func (m *Manager) reShadeSessionPath() string {
	return filepath.Join(m.dataDir, "reshade-sessions", "session.json")
}

func (m *Manager) readReShadeSession() (*reShadeSessionDocument, error) {
	contents, err := os.ReadFile(m.reShadeSessionPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var document reShadeSessionDocument
	if err := json.Unmarshal(contents, &document); err != nil {
		return nil, err
	}
	if document.Version != reShadeSessionVersion {
		return nil, fmt.Errorf("unsupported coordinated ReShade session version %d", document.Version)
	}
	return &document, nil
}

func (m *Manager) writeReShadeSession(document reShadeSessionDocument) error {
	contents, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	path := m.reShadeSessionPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".session-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(contents); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(tempPath, path)
}

func (m *Manager) removeReShadeSession(document reShadeSessionDocument, archive bool) error {
	root := filepath.Join(m.dataDir, "reshade-sessions", document.State.ID)
	if archive {
		archiveRoot := filepath.Join(m.dataDir, "archives", "reshade-sessions",
			fmt.Sprintf("%d-%s", m.now().UnixNano(), document.State.ID))
		if err := os.MkdirAll(filepath.Dir(archiveRoot), 0o755); err != nil {
			return err
		}
		if err := os.Rename(root, archiveRoot); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	} else if err := os.RemoveAll(root); err != nil {
		return err
	}
	if err := os.Remove(m.reShadeSessionPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
