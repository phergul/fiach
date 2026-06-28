package goosemigrations

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/fileops"
)

const (
	documentVersionV1 = 1
	documentVersionV2 = 2
	outputKindCopied  = "copied"
	winningSourceMod  = "mod"
)

type legacyManifestDocument struct {
	Version            int                                `json:"version"`
	AddedFiles         []legacyAddedFile                  `json:"addedFiles"`
	ReplacedFiles      []legacyReplacedFile               `json:"replacedFiles"`
	CreatedDirectories []legacyCreatedDirectory           `json:"createdDirectories"`
	Files              map[string]legacyManifestFileEntry `json:"files,omitempty"`
}

type legacyManifestFileEntry struct {
	GameRelativePath   string `json:"gameRelativePath"`
	OutputKind         string `json:"outputKind"`
	BaselineExists     bool   `json:"baselineExists"`
	BaselineSHA256     string `json:"baselineSha256,omitempty"`
	BaselineSizeBytes  int64  `json:"baselineSizeBytes,omitempty"`
	BaselineBackupPath string `json:"baselineBackupPath,omitempty"`
	AppliedExists      bool   `json:"appliedExists"`
	AppliedSHA256      string `json:"appliedSha256,omitempty"`
	AppliedSizeBytes   int64  `json:"appliedSizeBytes,omitempty"`
	WinningModID       *int64 `json:"winningModId,omitempty"`
	WinningLoadOrder   *int64 `json:"winningLoadOrder,omitempty"`
}

type legacyAddedFile struct {
	Mod        legacyMod `json:"mod"`
	TargetPath string    `json:"targetPath"`
	SHA256     string    `json:"sha256"`
	SizeBytes  int64     `json:"sizeBytes"`
}

type legacyReplacedFile struct {
	Mod             legacyMod `json:"mod"`
	TargetPath      string    `json:"targetPath"`
	SHA256          string    `json:"sha256"`
	SizeBytes       int64     `json:"sizeBytes"`
	BackupPath      string    `json:"backupPath"`
	BackupSHA256    string    `json:"backupSha256"`
	BackupSizeBytes int64     `json:"backupSizeBytes"`
}

type legacyCreatedDirectory struct {
	Mod        legacyMod `json:"mod"`
	TargetPath string    `json:"targetPath"`
}

type legacyMod struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type legacyFileStateRow struct {
	GameRelativePath   string
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
}

type legacyCreatedDirectoryRow struct {
	GameRelativePath string
	ModID            *int64
	ModName          *string
}

func decodeLegacyManifest(documentJSON string) (legacyManifestDocument, error) {
	documentJSON = strings.TrimSpace(documentJSON)
	if documentJSON == "" {
		return legacyManifestDocument{}, fmt.Errorf("manifest JSON is required")
	}

	var document legacyManifestDocument
	if err := json.Unmarshal([]byte(documentJSON), &document); err != nil {
		return legacyManifestDocument{}, fmt.Errorf("decode manifest JSON: %w", err)
	}
	if document.Version != documentVersionV1 && document.Version != documentVersionV2 {
		return legacyManifestDocument{}, fmt.Errorf("unsupported manifest version %d", document.Version)
	}

	if document.AddedFiles == nil {
		document.AddedFiles = []legacyAddedFile{}
	}
	if document.ReplacedFiles == nil {
		document.ReplacedFiles = []legacyReplacedFile{}
	}
	if document.CreatedDirectories == nil {
		document.CreatedDirectories = []legacyCreatedDirectory{}
	}
	if document.Files == nil {
		document.Files = map[string]legacyManifestFileEntry{}
	}

	return document, nil
}

func fileStatesFromLegacyManifest(document legacyManifestDocument, installPath string, appliedAt string) ([]legacyFileStateRow, error) {
	if len(document.Files) > 0 {
		return fileStatesFromManifestFiles(document.Files, appliedAt), nil
	}

	states := make([]legacyFileStateRow, 0, len(document.AddedFiles)+len(document.ReplacedFiles))
	seen := map[string]struct{}{}

	for _, entry := range document.AddedFiles {
		state, err := fileStateFromAddedFile(entry, installPath, appliedAt)
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
		state, err := fileStateFromReplacedFile(entry, installPath, appliedAt)
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

func createdDirectoriesFromLegacyManifest(document legacyManifestDocument, installPath string) ([]legacyCreatedDirectoryRow, error) {
	rows := make([]legacyCreatedDirectoryRow, 0, len(document.CreatedDirectories))
	seen := map[string]struct{}{}

	for _, entry := range document.CreatedDirectories {
		gameRelativePath, err := absoluteToGameRelativePath(installPath, entry.TargetPath)
		if err != nil {
			return nil, fmt.Errorf("created directory %q: %w", entry.TargetPath, err)
		}
		key := deployment.CanonicalGameRelativePath(gameRelativePath)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		row := legacyCreatedDirectoryRow{
			GameRelativePath: gameRelativePath,
		}
		if entry.Mod.ID > 0 {
			modID := entry.Mod.ID
			row.ModID = &modID
		}
		if strings.TrimSpace(entry.Mod.Name) != "" {
			modName := entry.Mod.Name
			row.ModName = &modName
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func fileStatesFromManifestFiles(files map[string]legacyManifestFileEntry, appliedAt string) []legacyFileStateRow {
	states := make([]legacyFileStateRow, 0, len(files))
	for _, entry := range files {
		states = append(states, persistedFileStateFromManifestEntry(entry, appliedAt))
	}

	return states
}

func fileStateFromAddedFile(entry legacyAddedFile, installPath string, appliedAt string) (legacyFileStateRow, error) {
	gameRelativePath, err := absoluteToGameRelativePath(installPath, entry.TargetPath)
	if err != nil {
		return legacyFileStateRow{}, fmt.Errorf("added file %q: %w", entry.TargetPath, err)
	}

	appliedSHA256 := entry.SHA256
	appliedSizeBytes := entry.SizeBytes
	winningModID := entry.Mod.ID
	winningSourceKind := winningSourceMod
	winningSourceID := strconv.FormatInt(entry.Mod.ID, 10)

	return legacyFileStateRow{
		GameRelativePath:  gameRelativePath,
		BaselineExists:    false,
		AppliedExists:     true,
		AppliedSHA256:     &appliedSHA256,
		AppliedSizeBytes:  &appliedSizeBytes,
		WinningSourceKind: &winningSourceKind,
		WinningSourceID:   &winningSourceID,
		WinningModID:      &winningModID,
		OutputKind:        outputKindCopied,
	}, nil
}

func fileStateFromReplacedFile(entry legacyReplacedFile, installPath string, appliedAt string) (legacyFileStateRow, error) {
	gameRelativePath, err := absoluteToGameRelativePath(installPath, entry.TargetPath)
	if err != nil {
		return legacyFileStateRow{}, fmt.Errorf("replaced file %q: %w", entry.TargetPath, err)
	}

	appliedSHA256 := entry.SHA256
	appliedSizeBytes := entry.SizeBytes
	baselineSHA256 := entry.BackupSHA256
	baselineSizeBytes := entry.BackupSizeBytes
	baselineBackupPath := entry.BackupPath
	winningModID := entry.Mod.ID
	winningSourceKind := winningSourceMod
	winningSourceID := strconv.FormatInt(entry.Mod.ID, 10)

	return legacyFileStateRow{
		GameRelativePath:   gameRelativePath,
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
		OutputKind:         outputKindCopied,
	}, nil
}

func persistedFileStateFromManifestEntry(entry legacyManifestFileEntry, appliedAt string) legacyFileStateRow {
	state := legacyFileStateRow{
		GameRelativePath: entry.GameRelativePath,
		BaselineExists:   entry.BaselineExists,
		AppliedExists:    entry.AppliedExists,
		OutputKind:       entry.OutputKind,
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
		winningSourceKind := winningSourceMod
		winningSourceID := strconv.FormatInt(winningModID, 10)
		state.WinningSourceKind = &winningSourceKind
		state.WinningSourceID = &winningSourceID
	}
	if entry.WinningLoadOrder != nil {
		winningLoadOrder := *entry.WinningLoadOrder
		state.WinningLoadOrder = &winningLoadOrder
	}

	if state.OutputKind == "" {
		state.OutputKind = outputKindCopied
	}
	if state.GameRelativePath == "" {
		state.GameRelativePath = entry.GameRelativePath
	}

	return state
}

func absoluteToGameRelativePath(installPath string, absolutePath string) (string, error) {
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

func filepathToSlash(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}
