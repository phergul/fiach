package optiscaler

import "time"

const (
	ManifestVersion = 1
	JournalVersion  = 1
	WarningVersion  = "online-anticheat-v1"
)

type GraphicsAPI string

const (
	GraphicsAPIDirectX GraphicsAPI = "directx"
	GraphicsAPIVulkan  GraphicsAPI = "vulkan"
)

type Action string

const (
	ActionInstall   Action = "install"
	ActionAdopt     Action = "adopt"
	ActionUpdate    Action = "update"
	ActionRepair    Action = "repair"
	ActionUninstall Action = "uninstall"
)

type Ownership string

const (
	OwnershipOptiScaler Ownership = "optiscaler"
	OwnershipReShade    Ownership = "reshade"
	OwnershipUnknown    Ownership = "unknown"
)

type Release struct {
	Tag       string `json:"tag"`
	Version   string `json:"version"`
	AssetName string `json:"assetName"`
	URL       string `json:"url"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
}

type Candidate struct {
	TargetRelativePath     string   `json:"targetRelativePath"`
	ExecutableRelativePath string   `json:"executableRelativePath"`
	ExecutableName         string   `json:"executableName"`
	Evidence               []string `json:"evidence"`
	Managed                bool     `json:"managed"`
	HasOptiScaler          bool     `json:"hasOptiScaler"`
	HasReShade             bool     `json:"hasReShade"`
}

type ManagedFile struct {
	RelativePath string `json:"relativePath"`
	SHA256       string `json:"sha256"`
	SizeBytes    int64  `json:"sizeBytes"`
	BackupPath   string `json:"backupPath,omitempty"`
	BackupSHA256 string `json:"backupSha256,omitempty"`
	BackupSize   int64  `json:"backupSizeBytes,omitempty"`
	Ownership    string `json:"ownership,omitempty"`
}

type ManagedConfig struct {
	LoadReShade       bool    `json:"loadReShade"`
	DXGISpoofing      bool    `json:"dxgiSpoofing"`
	TargetProcessName *string `json:"targetProcessName"`
	CheckForUpdate    bool    `json:"checkForUpdate"`
}

type Manifest struct {
	Version                    int           `json:"version"`
	Files                      []ManagedFile `json:"files"`
	Config                     ManagedConfig `json:"config"`
	OriginalReShadeProxy       *string       `json:"originalReShadeProxy,omitempty"`
	HasPreAdoptionRollbackData bool          `json:"hasPreAdoptionRollbackData"`
}

type Request struct {
	Action                   Action      `json:"action"`
	GameID                   int64       `json:"gameId"`
	TargetRelativePath       string      `json:"targetRelativePath"`
	ExecutableRelativePath   string      `json:"executableRelativePath"`
	GraphicsAPI              GraphicsAPI `json:"graphicsApi"`
	ProxyFilename            string      `json:"proxyFilename"`
	DXGISpoofing             bool        `json:"dxgiSpoofing"`
	ProcessFilter            *string     `json:"processFilter"`
	AcknowledgeWarning       bool        `json:"acknowledgeWarning"`
	BackupAndContinue        bool        `json:"backupAndContinue"`
	EnableReShadeCoexistence bool        `json:"enableReShadeCoexistence"`
}

type Operation struct {
	Type       string `json:"type"`
	SourcePath string `json:"sourcePath,omitempty"`
	TargetPath string `json:"targetPath"`
	SHA256     string `json:"sha256,omitempty"`
	SizeBytes  int64  `json:"sizeBytes,omitempty"`
}

type Drift struct {
	RelativePath string `json:"relativePath"`
	ExpectedHash string `json:"expectedHash"`
	ActualHash   string `json:"actualHash,omitempty"`
	Missing      bool   `json:"missing"`
}

type Preview struct {
	Request              Request     `json:"request"`
	Release              Release     `json:"release"`
	Operations           []Operation `json:"operations"`
	ConfigurationChanges []string    `json:"configurationChanges"`
	Warnings             []string    `json:"warnings"`
	Conflicts            []string    `json:"conflicts"`
	Drift                []Drift     `json:"drift"`
	PreviewHash          string      `json:"previewHash"`
	CanApply             bool        `json:"canApply"`
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

var SupportedProxyFilenames = []string{
	"dxgi.dll", "winmm.dll", "d3d12.dll", "dbghelp.dll",
	"version.dll", "wininet.dll", "winhttp.dll", "OptiScaler.asi",
}
