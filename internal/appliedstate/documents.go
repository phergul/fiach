package appliedstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/phergul/mod-manager/internal/operationplan"
)

const DocumentVersion = 1

type ManifestDocument struct {
	Version            int                `json:"version"`
	AddedFiles         []AddedFile        `json:"addedFiles"`
	ReplacedFiles      []ReplacedFile     `json:"replacedFiles"`
	CreatedDirectories []CreatedDirectory `json:"createdDirectories"`
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

type EncodedProfileSnapshot struct {
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

func EncodeManifest(document ManifestDocument) (string, error) {
	encoded, err := json.Marshal(document)
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}

func EncodeProfileSnapshot(document ProfileSnapshotDocument) (EncodedProfileSnapshot, error) {
	encoded, err := json.Marshal(document)
	if err != nil {
		return EncodedProfileSnapshot{}, err
	}

	sum := sha256.Sum256(encoded)
	return EncodedProfileSnapshot{
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
