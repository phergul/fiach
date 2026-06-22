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
	ActionInstall          Action = "install"
	ActionAdopt            Action = "adopt"
	ActionUpdate           Action = "update"
	ActionRepair           Action = "repair"
	ActionUninstall        Action = "uninstall"
	ActionConfigureContent Action = "configure_content"
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

type VariantProvenance string

const (
	VariantProvenanceVerified     VariantProvenance = "verified"
	VariantProvenanceUserDeclared VariantProvenance = "user_declared"
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
	RelativePath string          `json:"relativePath"`
	SHA256       string          `json:"sha256"`
	SizeBytes    int64           `json:"sizeBytes"`
	Ownership    Ownership       `json:"ownership"`
	Role         PathRole        `json:"role,omitempty"`
	Source       *ContentSource  `json:"source,omitempty"`
	Sources      []ContentSource `json:"sources,omitempty"`
	BackupPath   *string         `json:"backupPath,omitempty"`
	BackupSHA256 *string         `json:"backupSha256,omitempty"`
	BackupSize   *int64          `json:"backupSizeBytes,omitempty"`
}

type ContentSourceKind string

const (
	ContentSourceRuntime       ContentSourceKind = "runtime"
	ContentSourceEffectPackage ContentSourceKind = "effect_package"
	ContentSourceAddon         ContentSourceKind = "addon"
)

type ContentSource struct {
	Kind          ContentSourceKind `json:"kind"`
	ID            string            `json:"id,omitempty"`
	Name          string            `json:"name,omitempty"`
	RepositoryURL string            `json:"repositoryUrl,omitempty"`
	DownloadURL   string            `json:"downloadUrl,omitempty"`
	ArchiveSHA256 string            `json:"archiveSha256,omitempty"`
	ArchiveSize   int64             `json:"archiveSizeBytes,omitempty"`
	Shared        bool              `json:"shared,omitempty"`
}

type PathRole string

const (
	PathRoleRuntime       PathRole = "runtime"
	PathRoleConfiguration PathRole = "configuration"
	PathRolePreset        PathRole = "preset"
	PathRoleLog           PathRole = "log"
	PathRoleBackup        PathRole = "backup"
	PathRoleEffects       PathRole = "effects"
	PathRoleTextures      PathRole = "textures"
	PathRoleAddons        PathRole = "addons"
	PathRoleScreenshots   PathRole = "screenshots"
)

type UserContent struct {
	Path          string   `json:"path"`
	Role          PathRole `json:"role"`
	SHA256        string   `json:"sha256,omitempty"`
	SizeBytes     int64    `json:"sizeBytes,omitempty"`
	Exists        bool     `json:"exists"`
	External      bool     `json:"external"`
	Directory     bool     `json:"directory"`
	InventoryOnly bool     `json:"inventoryOnly"`
}

type Manifest struct {
	Version                    int               `json:"version"`
	Files                      []ManagedFile     `json:"files"`
	HasPreAdoptionRollbackData bool              `json:"hasPreAdoptionRollbackData"`
	VariantProvenance          VariantProvenance `json:"variantProvenance,omitempty"`
	UserContent                []UserContent     `json:"userContent,omitempty"`
}

type Request struct {
	Action                    Action         `json:"action"`
	GameID                    int64          `json:"gameId"`
	TargetRelativePath        string         `json:"targetRelativePath"`
	ExecutableRelativePath    string         `json:"executableRelativePath"`
	RenderingAPI              RenderingAPI   `json:"renderingApi"`
	ProxyFilename             string         `json:"proxyFilename"`
	Architecture              Architecture   `json:"architecture"`
	BuildVariant              BuildVariant   `json:"buildVariant"`
	BackupAndContinue         bool           `json:"backupAndContinue"`
	SinglePlayerAcknowledged  bool           `json:"singlePlayerAcknowledged"`
	AntiCheatRiskAcknowledged bool           `json:"antiCheatRiskAcknowledged"`
	Content                   ContentRequest `json:"content,omitempty"`
}

type ContentRequest struct {
	EffectPackages []EffectPackageSelection `json:"effectPackages,omitempty"`
	Addons         []AddonSelection         `json:"addons,omitempty"`
}

type EffectPackageSelection struct {
	ID          string   `json:"id"`
	EffectFiles []string `json:"effectFiles,omitempty"`
}

type AddonSelection struct {
	ID string `json:"id"`
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

type UserContentDrift struct {
	Path         string   `json:"path"`
	Role         PathRole `json:"role"`
	ExpectedHash string   `json:"expectedHash,omitempty"`
	ActualHash   string   `json:"actualHash,omitempty"`
	Missing      bool     `json:"missing"`
	External     bool     `json:"external"`
}

type PathImpact struct {
	Path             string    `json:"path"`
	Role             PathRole  `json:"role"`
	Action           string    `json:"action"`
	Ownership        Ownership `json:"ownership"`
	Exists           bool      `json:"exists"`
	Blocking         bool      `json:"blocking"`
	PreservationOnly bool      `json:"preservationOnly"`
}

type Preview struct {
	Request          Request            `json:"request"`
	Operations       []Operation        `json:"operations"`
	PathImpacts      []PathImpact       `json:"pathImpacts"`
	Warnings         []string           `json:"warnings"`
	Conflicts        []string           `json:"conflicts"`
	Drift            []Drift            `json:"drift"`
	UserContentDrift []UserContentDrift `json:"userContentDrift"`
	DesiredTarget    *TargetState       `json:"desiredTarget,omitempty"`
	PreviewHash      string             `json:"previewHash"`
	CanApply         bool               `json:"canApply"`
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
	VariantProvenance      VariantProvenance
	RuntimeVersion         string
	Provenance             InstallerProvenance
	ManagementOrigin       string
	Status                 ManagementStatus
	CreatedAt              string
	UpdatedAt              string
	LastVerifiedAt         *string
}

type APIProxyOptions struct {
	RenderingAPI RenderingAPI `json:"renderingApi"`
	Proxies      []string     `json:"proxies"`
}

type ProxyEvidence struct {
	Filename       string       `json:"filename"`
	Exists         bool         `json:"exists"`
	IsReShade      bool         `json:"isReShade"`
	Architecture   Architecture `json:"architecture,omitempty"`
	RuntimeVersion string       `json:"runtimeVersion,omitempty"`
	Conflict       string       `json:"conflict,omitempty"`
}

type Candidate struct {
	TargetRelativePath     string            `json:"targetRelativePath"`
	ExecutableRelativePath string            `json:"executableRelativePath"`
	Architecture           Architecture      `json:"architecture"`
	APIOptions             []APIProxyOptions `json:"apiOptions"`
	ProxyEvidence          []ProxyEvidence   `json:"proxyEvidence"`
	Conflicts              []string          `json:"conflicts"`
}

type DiscoveryWarning struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type DiscoveryResult struct {
	Candidates []Candidate        `json:"candidates"`
	Warnings   []DiscoveryWarning `json:"warnings"`
}

type ContentCatalogue struct {
	Effects []EffectPackage `json:"effects"`
	Addons  []AddonPackage  `json:"addons"`
	Cached  bool            `json:"cached"`
}

type EffectPackage struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	InstallPath        string   `json:"installPath"`
	TextureInstallPath string   `json:"textureInstallPath"`
	DownloadURL        string   `json:"downloadUrl"`
	RepositoryURL      string   `json:"repositoryUrl"`
	Required           bool     `json:"required"`
	Enabled            bool     `json:"enabled"`
	Modifiable         bool     `json:"modifiable"`
	EffectFiles        []string `json:"effectFiles"`
	DenyEffectFiles    []string `json:"denyEffectFiles"`
}

type AddonPackage struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	EffectInstallPath string `json:"effectInstallPath,omitempty"`
	DownloadURL       string `json:"downloadUrl,omitempty"`
	DownloadURL32     string `json:"downloadUrl32,omitempty"`
	DownloadURL64     string `json:"downloadUrl64,omitempty"`
	RepositoryURL     string `json:"repositoryUrl"`
}

type PresetInspectionRequest struct {
	GameID             int64  `json:"gameId"`
	TargetRelativePath string `json:"targetRelativePath"`
	PresetPath         string `json:"presetPath"`
}

type PresetInspectionResult struct {
	ReferencedEffects []string               `json:"referencedEffects"`
	Recommendations   []PresetRecommendation `json:"recommendations"`
	MissingEffects    []string               `json:"missingEffects"`
	Warnings          []string               `json:"warnings"`
}

type PresetRecommendation struct {
	PackageID   string   `json:"packageId"`
	PackageName string   `json:"packageName"`
	EffectFiles []string `json:"effectFiles"`
}

type InstallerReleaseStatus struct {
	Version   string           `json:"version"`
	Variant   InstallerVariant `json:"variant"`
	AssetName string           `json:"assetName"`
	URL       string           `json:"url"`
	Digest    *string          `json:"digest,omitempty"`
	Size      *int64           `json:"sizeBytes,omitempty"`
	Cached    bool             `json:"cached"`
	Error     string           `json:"error,omitempty"`
}

type InstallerStatus struct {
	Standard InstallerReleaseStatus `json:"standard"`
	Addon    InstallerReleaseStatus `json:"addon"`
}
