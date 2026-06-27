package appliedstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/phergul/fiach/internal/operationplan"
)

const (
	DocumentVersionV1 = 1
	DocumentVersionV2 = 2
	DocumentVersion   = DocumentVersionV2
)

type ManifestDocument struct {
	Version            int                          `json:"version"`
	AddedFiles         []AddedFile                  `json:"addedFiles"`
	ReplacedFiles      []ReplacedFile               `json:"replacedFiles"`
	CreatedDirectories []CreatedDirectory           `json:"createdDirectories"`
	Files              map[string]ManifestFileEntry `json:"files,omitempty"`
}

type ManifestFileEntry struct {
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

type AddedFile struct {
	OperationIndex int    `json:"operationIndex"`
	Mod            Mod    `json:"mod"`
	SourcePath     string `json:"sourcePath"`
	TargetPath     string `json:"targetPath"`
	SHA256         string `json:"sha256"`
	SizeBytes      int64  `json:"sizeBytes"`
}

type ReplacedFile struct {
	OperationIndex  int    `json:"operationIndex"`
	Mod             Mod    `json:"mod"`
	SourcePath      string `json:"sourcePath"`
	TargetPath      string `json:"targetPath"`
	SHA256          string `json:"sha256"`
	SizeBytes       int64  `json:"sizeBytes"`
	BackupPath      string `json:"backupPath"`
	BackupSHA256    string `json:"backupSha256"`
	BackupSizeBytes int64  `json:"backupSizeBytes"`
}

type CreatedDirectory struct {
	OperationIndex int    `json:"operationIndex"`
	Mod            Mod    `json:"mod"`
	TargetPath     string `json:"targetPath"`
}

type ProfileSnapshotDocument struct {
	Version    int                 `json:"version"`
	CanApply   bool                `json:"canApply"`
	Operations []SnapshotOperation `json:"operations"`
}

type ProfileCompositionDocument struct {
	Version   int                     `json:"version"`
	ProfileID int64                   `json:"profileId"`
	Mods      []ProfileCompositionMod `json:"mods"`
}

type ProfileCompositionMod struct {
	ModID            int64  `json:"modId"`
	Enabled          bool   `json:"enabled"`
	LoadOrder        int64  `json:"loadOrder"`
	SourcePath       string `json:"sourcePath"`
	PackageUpdatedAt string `json:"packageUpdatedAt"`
}

type SnapshotOperation struct {
	OperationIndex int                         `json:"operationIndex"`
	Type           operationplan.OperationType `json:"type"`
	Mod            Mod                         `json:"mod"`
	SourcePath     *string                     `json:"sourcePath"`
	TargetPath     string                      `json:"targetPath"`
	BackupPath     *string                     `json:"backupPath"`
}

type Mod struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type EncodedSnapshot struct {
	JSON string
	Hash string
}

func BuildManifestDocument(manifest operationplan.AppliedOperationManifest) ManifestDocument {
	document := ManifestDocument{
		Version:            DocumentVersion,
		AddedFiles:         make([]AddedFile, 0, len(manifest.AddedFiles)),
		ReplacedFiles:      make([]ReplacedFile, 0, len(manifest.ReplacedFiles)),
		CreatedDirectories: make([]CreatedDirectory, 0, len(manifest.CreatedDirectories)),
	}

	for _, entry := range manifest.AddedFiles {
		document.AddedFiles = append(document.AddedFiles, AddedFile{
			OperationIndex: entry.OperationIndex,
			Mod:            modDocument(entry.Mod),
			SourcePath:     entry.SourcePath,
			TargetPath:     entry.TargetPath,
			SHA256:         entry.SHA256,
			SizeBytes:      entry.SizeBytes,
		})
	}
	for _, entry := range manifest.ReplacedFiles {
		document.ReplacedFiles = append(document.ReplacedFiles, ReplacedFile{
			OperationIndex:  entry.OperationIndex,
			Mod:             modDocument(entry.Mod),
			SourcePath:      entry.SourcePath,
			TargetPath:      entry.TargetPath,
			SHA256:          entry.SHA256,
			SizeBytes:       entry.SizeBytes,
			BackupPath:      entry.BackupPath,
			BackupSHA256:    entry.BackupSHA256,
			BackupSizeBytes: entry.BackupSizeBytes,
		})
	}
	for _, entry := range manifest.CreatedDirectories {
		document.CreatedDirectories = append(document.CreatedDirectories, CreatedDirectory{
			OperationIndex: entry.OperationIndex,
			Mod:            modDocument(entry.Mod),
			TargetPath:     entry.TargetPath,
		})
	}

	return document
}

func BuildProfileSnapshotDocument(plan operationplan.OperationPlan) ProfileSnapshotDocument {
	document := ProfileSnapshotDocument{
		Version:    DocumentVersion,
		CanApply:   plan.CanApply,
		Operations: make([]SnapshotOperation, 0, len(plan.Operations)),
	}

	for index, operation := range plan.Operations {
		document.Operations = append(document.Operations, SnapshotOperation{
			OperationIndex: index,
			Type:           operation.Type,
			Mod:            modDocument(operation.Mod),
			SourcePath:     copyStringPtr(operation.SourcePath),
			TargetPath:     operation.TargetPath,
			BackupPath:     copyStringPtr(operation.BackupPath),
		})
	}

	return document
}

func BuildProfileCompositionDocument(profileID int64, mods []ProfileCompositionMod) ProfileCompositionDocument {
	copiedMods := make([]ProfileCompositionMod, len(mods))
	copy(copiedMods, mods)
	sort.SliceStable(copiedMods, func(i int, j int) bool {
		if copiedMods[i].LoadOrder != copiedMods[j].LoadOrder {
			return copiedMods[i].LoadOrder < copiedMods[j].LoadOrder
		}

		return copiedMods[i].ModID < copiedMods[j].ModID
	})

	return ProfileCompositionDocument{
		Version:   DocumentVersion,
		ProfileID: profileID,
		Mods:      copiedMods,
	}
}

func EncodeManifest(document ManifestDocument) (string, error) {
	encoded, err := json.Marshal(document)
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}

func DecodeManifest(documentJSON string) (ManifestDocument, error) {
	documentJSON = strings.TrimSpace(documentJSON)
	if documentJSON == "" {
		return ManifestDocument{}, fmt.Errorf("manifest JSON is required")
	}

	var document ManifestDocument
	if err := json.Unmarshal([]byte(documentJSON), &document); err != nil {
		return ManifestDocument{}, fmt.Errorf("decode manifest JSON: %w", err)
	}
	if document.Version != DocumentVersionV1 && document.Version != DocumentVersionV2 {
		return ManifestDocument{}, fmt.Errorf("unsupported manifest version %d", document.Version)
	}

	if document.AddedFiles == nil {
		document.AddedFiles = []AddedFile{}
	}
	if document.ReplacedFiles == nil {
		document.ReplacedFiles = []ReplacedFile{}
	}
	if document.CreatedDirectories == nil {
		document.CreatedDirectories = []CreatedDirectory{}
	}
	if document.Files == nil {
		document.Files = map[string]ManifestFileEntry{}
	}

	return document, nil
}

func EncodeProfileSnapshot(document ProfileSnapshotDocument) (EncodedSnapshot, error) {
	encoded, err := json.Marshal(document)
	if err != nil {
		return EncodedSnapshot{}, err
	}

	sum := sha256.Sum256(encoded)
	return EncodedSnapshot{
		JSON: string(encoded),
		Hash: hex.EncodeToString(sum[:]),
	}, nil
}

func EncodeProfileCompositionSnapshot(document ProfileCompositionDocument) (snapshot EncodedSnapshot, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("marshal profile composition snapshot: %w", err)
		}
	}()

	encoded, err := json.Marshal(document)
	if err != nil {
		return EncodedSnapshot{}, err
	}

	sum := sha256.Sum256(encoded)
	return EncodedSnapshot{
		JSON: string(encoded),
		Hash: hex.EncodeToString(sum[:]),
	}, nil
}

func modDocument(mod operationplan.ModContext) Mod {
	return Mod{
		ID:   mod.ModID,
		Name: mod.ModName,
	}
}

func copyStringPtr(value *string) *string {
	if value == nil {
		return nil
	}

	copied := *value
	return &copied
}
