package dto

import (
	"github.com/phergul/fiach/internal/injection"
	"github.com/phergul/fiach/internal/reshade"
)

type ReShadeDetectionStatus string

const (
	ReShadeDetectionStatusInstalled    ReShadeDetectionStatus = "installed"
	ReShadeDetectionStatusNotInstalled ReShadeDetectionStatus = "not_installed"
	ReShadeDetectionStatusUnsupported  ReShadeDetectionStatus = "unsupported"
)

type DetectedReShadeTarget struct {
	Path             string
	Executables      []string
	ManagementStatus reshade.ManagementStatus
}

type ReShadeDetectionResult struct {
	Status            ReShadeDetectionStatus
	Targets           []DetectedReShadeTarget
	UnsupportedReason *string
}

type ReShadeRequest = reshade.Request
type ReShadePreview = reshade.Preview
type ReShadeApplyResult = reshade.ApplyResult
type ReShadeRecoveryState = reshade.RecoveryState
type ReShadeTarget = reshade.ManagedTarget
type ReShadeDiscoveryResult = reshade.DiscoveryResult
type ReShadeContentCatalogue = reshade.ContentCatalogue
type ReShadePresetInspectionResult = reshade.PresetInspectionResult
type ReShadeInstallerStatus = reshade.InstallerStatus

type ReShadeChainTarget struct {
	GameID                 int64
	TargetRelativePath     string
	ExecutableRelativePath string
	APIFamily              injection.APIFamily
	PrimaryOwner           injection.Owner
	PrimaryProxyFilename   string
	Status                 injection.Status
	OptiScaler             *ReShadeOptiScalerChainState
	ReShade                *ReShadeChainState
}

type ReShadeOptiScalerChainState struct {
	ProxyFilename string
	Status        string
}

type ReShadeChainState struct {
	PreferredProxyFilename string
	ActiveRuntimeFilename  string
	Status                 string
}
