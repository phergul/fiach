package reshade

import (
	"time"

	"github.com/phergul/fiach/internal/filetxn"
)

const (
	ManifestVersion = 1
	JournalVersion  = 1
)

type Action string

const (
	ActionInstall   Action = "install"
	ActionAdopt     Action = "adopt"
	ActionUpdate    Action = "update"
	ActionRepair    Action = "repair"
	ActionUninstall Action = "uninstall"
)

type RenderingAPI string

const (
	RenderingAPID3D9  RenderingAPI = "d3d9"
	RenderingAPID3D10 RenderingAPI = "d3d10"
	RenderingAPID3D11 RenderingAPI = "d3d11"
	RenderingAPID3D12 RenderingAPI = "d3d12"
)

type Architecture string

const (
	ArchitectureX86 Architecture = "x86"
	ArchitectureX64 Architecture = "x64"
)

type BuildVariant string

const (
	BuildVariantStandard BuildVariant = "standard"
	BuildVariantAddon    BuildVariant = "addon"
)

type Ownership string

const (
	OwnershipManaged Ownership = "managed"
	OwnershipAdopted Ownership = "adopted"
	OwnershipUser    Ownership = "user"
	OwnershipForeign Ownership = "foreign"
)

type ManagementStatus string

const (
	ManagementStatusUnmanaged            ManagementStatus = "unmanaged"
	ManagementStatusManaged              ManagementStatus = "managed"
	ManagementStatusDrifted              ManagementStatus = "drifted"
	ManagementStatusRecoveryRequired     ManagementStatus = "recovery_required"
	ManagementStatusIncompatibleManifest ManagementStatus = "incompatible_manifest"
)

type InstallerProvenance struct {
	Tag       *string `json:"tag,omitempty"`
	AssetName *string `json:"assetName,omitempty"`
	URL       *string `json:"url,omitempty"`
	Digest    *string `json:"digest,omitempty"`
	Size      *int64  `json:"size,omitempty"`
}

type ManagedFile struct {
	RelativePath string    `json:"relativePath"`
	SHA256       string    `json:"sha256"`
	SizeBytes    int64     `json:"sizeBytes"`
	Ownership    Ownership `json:"ownership"`
	BackupPath   *string   `json:"backupPath,omitempty"`
	BackupSHA256 *string   `json:"backupSha256,omitempty"`
	BackupSize   *int64    `json:"backupSizeBytes,omitempty"`
}

type Manifest struct {
	Version                    int           `json:"version"`
	Files                      []ManagedFile `json:"files"`
	HasPreAdoptionRollbackData bool          `json:"hasPreAdoptionRollbackData"`
}

type Request struct {
	Action                 Action       `json:"action"`
	GameID                 int64        `json:"gameId"`
	TargetRelativePath     string       `json:"targetRelativePath"`
	ExecutableRelativePath string       `json:"executableRelativePath"`
	RenderingAPI           RenderingAPI `json:"renderingApi"`
	ProxyFilename          string       `json:"proxyFilename"`
	Architecture           Architecture `json:"architecture"`
	BuildVariant           BuildVariant `json:"buildVariant"`
	BackupAndContinue      bool         `json:"backupAndContinue"`
}

type TargetState struct {
	RuntimeVersion   string              `json:"runtimeVersion"`
	Provenance       InstallerProvenance `json:"provenance"`
	ManagementOrigin string              `json:"managementOrigin"`
	Manifest         Manifest            `json:"manifest"`
}

type Operation = filetxn.Operation

type Drift struct {
	RelativePath string `json:"relativePath"`
	ExpectedHash string `json:"expectedHash"`
	ActualHash   string `json:"actualHash,omitempty"`
	Missing      bool   `json:"missing"`
}

type Preview struct {
	Request       Request      `json:"request"`
	Operations    []Operation  `json:"operations"`
	Warnings      []string     `json:"warnings"`
	Conflicts     []string     `json:"conflicts"`
	Drift         []Drift      `json:"drift"`
	DesiredTarget *TargetState `json:"desiredTarget,omitempty"`
	PreviewHash   string       `json:"previewHash"`
	CanApply      bool         `json:"canApply"`
}

type ApplyResult struct {
	Success    bool   `json:"success"`
	RolledBack bool   `json:"rolledBack"`
	Message    string `json:"message"`
}

type RecoveryState struct {
	Required   bool      `json:"required"`
	JournalID  string    `json:"journalId,omitempty"`
	GameID     int64     `json:"gameId,omitempty"`
	TargetPath string    `json:"targetPath,omitempty"`
	Action     Action    `json:"action,omitempty"`
	StartedAt  time.Time `json:"startedAt,omitempty"`
	Error      string    `json:"error,omitempty"`
}

type ManagedTarget struct {
	ID                     int64
	GameID                 int64
	TargetRelativePath     string
	ExecutableRelativePath string
	RenderingAPI           RenderingAPI
	ProxyFilename          string
	Architecture           Architecture
	BuildVariant           BuildVariant
	RuntimeVersion         string
	Provenance             InstallerProvenance
	ManagementOrigin       string
	Status                 ManagementStatus
	CreatedAt              string
	UpdatedAt              string
	LastVerifiedAt         *string
}
