package appliedstate

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/fileops"
)

const OutputKindCopied = "copied"

const WinningSourceKindMod = "mod"

type PersistedFileState struct {
	GameID             int64
	GameRelativePath   string
	ProfileID          int64
	BaselineExists     bool
	BaselineSHA256     *string
	BaselineSizeBytes  *int64
	BaselineBackupPath *string
	AppliedExists      bool
	AppliedSHA256      *string
	AppliedSizeBytes   *int64
	WinningSourceKind  *string
	WinningSourceID    *string
	WinningModID       *int64
	WinningLoadOrder   *int64
	OutputKind         string
	UserDecision       *string
	LastAppliedAt      string
}

func AbsoluteToGameRelativePath(installPath string, absolutePath string) (string, error) {
	cleanInstallPath, err := fileops.CleanAbsPath("game install path", installPath)
	if err != nil {
		return "", err
	}
	cleanAbsolutePath, err := fileops.CleanAbsPath("managed file path", absolutePath)
	if err != nil {
		return "", err
	}
	if err := fileops.RequirePathWithinRoot("managed file path", cleanAbsolutePath, cleanInstallPath); err != nil {
		return "", err
	}

	relativePath, err := filepath.Rel(cleanInstallPath, cleanAbsolutePath)
	if err != nil {
		return "", fmt.Errorf("resolve game-relative path: %w", err)
	}

	return filepathToSlash(relativePath), nil
}

func BuildFileStatesFromManifest(document ManifestDocument, installPath string, profileID int64, appliedAt string) ([]PersistedFileState, error) {
	if len(document.Files) > 0 {
		return fileStatesFromManifestFiles(document.Files, profileID, appliedAt), nil
	}

	states := make([]PersistedFileState, 0, len(document.AddedFiles)+len(document.ReplacedFiles))
	seen := map[string]struct{}{}

	for _, entry := range document.AddedFiles {
		state, err := fileStateFromAddedFile(entry, installPath, profileID, appliedAt)
		if err != nil {
			return nil, err
		}
		key := deployment.CanonicalGameRelativePath(state.GameRelativePath)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		states = append(states, state)
	}

	for _, entry := range document.ReplacedFiles {
		state, err := fileStateFromReplacedFile(entry, installPath, profileID, appliedAt)
		if err != nil {
			return nil, err
		}
		key := deployment.CanonicalGameRelativePath(state.GameRelativePath)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		states = append(states, state)
	}

	return states, nil
}

func BuildManifestFilesMap(fileStates []PersistedFileState) map[string]ManifestFileEntry {
	files := make(map[string]ManifestFileEntry, len(fileStates))
	for _, state := range fileStates {
		key := deployment.CanonicalGameRelativePath(state.GameRelativePath)
		files[key] = manifestFileEntryFromState(state)
	}

	return files
}

func AttachManifestFiles(document *ManifestDocument, fileStates []PersistedFileState) {
	if document == nil {
		return
	}

	document.Version = DocumentVersion
	document.Files = BuildManifestFilesMap(fileStates)
}

func fileStatesFromManifestFiles(files map[string]ManifestFileEntry, profileID int64, appliedAt string) []PersistedFileState {
	states := make([]PersistedFileState, 0, len(files))
	for _, entry := range files {
		state := persistedFileStateFromManifestEntry(entry, profileID, appliedAt)
		states = append(states, state)
	}

	return states
}

func fileStateFromAddedFile(entry AddedFile, installPath string, profileID int64, appliedAt string) (PersistedFileState, error) {
	gameRelativePath, err := AbsoluteToGameRelativePath(installPath, entry.TargetPath)
	if err != nil {
		return PersistedFileState{}, fmt.Errorf("added file %q: %w", entry.TargetPath, err)
	}

	appliedSHA256 := entry.SHA256
	appliedSizeBytes := entry.SizeBytes
	winningModID := entry.Mod.ID
	winningSourceKind := WinningSourceKindMod
	winningSourceID := strconv.FormatInt(entry.Mod.ID, 10)

	return PersistedFileState{
		GameRelativePath:  gameRelativePath,
		ProfileID:         profileID,
		BaselineExists:    false,
		AppliedExists:     true,
		AppliedSHA256:     &appliedSHA256,
		AppliedSizeBytes:  &appliedSizeBytes,
		WinningSourceKind: &winningSourceKind,
		WinningSourceID:   &winningSourceID,
		WinningModID:      &winningModID,
		OutputKind:        OutputKindCopied,
		LastAppliedAt:     appliedAt,
	}, nil
}

func fileStateFromReplacedFile(entry ReplacedFile, installPath string, profileID int64, appliedAt string) (PersistedFileState, error) {
	gameRelativePath, err := AbsoluteToGameRelativePath(installPath, entry.TargetPath)
	if err != nil {
		return PersistedFileState{}, fmt.Errorf("replaced file %q: %w", entry.TargetPath, err)
	}

	appliedSHA256 := entry.SHA256
	appliedSizeBytes := entry.SizeBytes
	baselineSHA256 := entry.BackupSHA256
	baselineSizeBytes := entry.BackupSizeBytes
	baselineBackupPath := entry.BackupPath
	winningModID := entry.Mod.ID
	winningSourceKind := WinningSourceKindMod
	winningSourceID := strconv.FormatInt(entry.Mod.ID, 10)

	return PersistedFileState{
		GameRelativePath:   gameRelativePath,
		ProfileID:          profileID,
		BaselineExists:     true,
		BaselineSHA256:     &baselineSHA256,
		BaselineSizeBytes:  &baselineSizeBytes,
		BaselineBackupPath: &baselineBackupPath,
		AppliedExists:      true,
		AppliedSHA256:      &appliedSHA256,
		AppliedSizeBytes:   &appliedSizeBytes,
		WinningSourceKind:  &winningSourceKind,
		WinningSourceID:    &winningSourceID,
		WinningModID:       &winningModID,
		OutputKind:         OutputKindCopied,
		LastAppliedAt:      appliedAt,
	}, nil
}

func persistedFileStateFromManifestEntry(entry ManifestFileEntry, profileID int64, appliedAt string) PersistedFileState {
	state := PersistedFileState{
		GameRelativePath: entry.GameRelativePath,
		ProfileID:        profileID,
		BaselineExists:   entry.BaselineExists,
		AppliedExists:    entry.AppliedExists,
		OutputKind:       entry.OutputKind,
		LastAppliedAt:    appliedAt,
	}
	if entry.BaselineSHA256 != "" {
		baselineSHA256 := entry.BaselineSHA256
		state.BaselineSHA256 = &baselineSHA256
	}
	if entry.BaselineSizeBytes != 0 {
		baselineSizeBytes := entry.BaselineSizeBytes
		state.BaselineSizeBytes = &baselineSizeBytes
	}
	if entry.BaselineBackupPath != "" {
		baselineBackupPath := entry.BaselineBackupPath
		state.BaselineBackupPath = &baselineBackupPath
	}
	if entry.AppliedSHA256 != "" {
		appliedSHA256 := entry.AppliedSHA256
		state.AppliedSHA256 = &appliedSHA256
	}
	if entry.AppliedSizeBytes != 0 {
		appliedSizeBytes := entry.AppliedSizeBytes
		state.AppliedSizeBytes = &appliedSizeBytes
	}
	if entry.WinningModID != nil {
		winningModID := *entry.WinningModID
		state.WinningModID = &winningModID
		winningSourceKind := WinningSourceKindMod
		winningSourceID := strconv.FormatInt(winningModID, 10)
		state.WinningSourceKind = &winningSourceKind
		state.WinningSourceID = &winningSourceID
	}
	if entry.WinningLoadOrder != nil {
		winningLoadOrder := *entry.WinningLoadOrder
		state.WinningLoadOrder = &winningLoadOrder
	}

	if state.OutputKind == "" {
		state.OutputKind = OutputKindCopied
	}
	if state.GameRelativePath == "" {
		state.GameRelativePath = entry.GameRelativePath
	}

	return state
}

func manifestFileEntryFromState(state PersistedFileState) ManifestFileEntry {
	entry := ManifestFileEntry{
		GameRelativePath: state.GameRelativePath,
		OutputKind:       state.OutputKind,
		BaselineExists:   state.BaselineExists,
		AppliedExists:    state.AppliedExists,
	}
	if state.BaselineSHA256 != nil {
		entry.BaselineSHA256 = *state.BaselineSHA256
	}
	if state.BaselineSizeBytes != nil {
		entry.BaselineSizeBytes = *state.BaselineSizeBytes
	}
	if state.BaselineBackupPath != nil {
		entry.BaselineBackupPath = *state.BaselineBackupPath
	}
	if state.AppliedSHA256 != nil {
		entry.AppliedSHA256 = *state.AppliedSHA256
	}
	if state.AppliedSizeBytes != nil {
		entry.AppliedSizeBytes = *state.AppliedSizeBytes
	}
	if state.WinningModID != nil {
		entry.WinningModID = state.WinningModID
	}
	if state.WinningLoadOrder != nil {
		entry.WinningLoadOrder = state.WinningLoadOrder
	}

	if entry.OutputKind == "" {
		entry.OutputKind = OutputKindCopied
	}

	return entry
}

func filepathToSlash(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}
