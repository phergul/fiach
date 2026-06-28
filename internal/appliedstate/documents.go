package appliedstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
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

type DeploymentProfileSnapshotDocument struct {
	Version     int    `json:"version"`
	PreviewHash string `json:"previewHash"`
	PlanMode    string `json:"planMode"`
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

type Mod struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type EncodedSnapshot struct {
	JSON string
	Hash string
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

func BuildDeploymentProfileSnapshotDocument(previewHash string, planMode string) DeploymentProfileSnapshotDocument {
	return DeploymentProfileSnapshotDocument{
		Version:     DocumentVersion,
		PreviewHash: previewHash,
		PlanMode:    planMode,
	}
}

func EncodeDeploymentProfileSnapshot(document DeploymentProfileSnapshotDocument) (EncodedSnapshot, error) {
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
