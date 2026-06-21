package dto

import "github.com/phergul/fiach/internal/reshade"

type ReShadeDetectionStatus string

const (
	ReShadeDetectionStatusInstalled    ReShadeDetectionStatus = "installed"
	ReShadeDetectionStatusNotInstalled ReShadeDetectionStatus = "not_installed"
	ReShadeDetectionStatusUnsupported  ReShadeDetectionStatus = "unsupported"
)

type ReShadeTarget struct {
	Path             string
	Executables      []string
	ManagementStatus reshade.ManagementStatus
}

type ReShadeDetectionResult struct {
	Status            ReShadeDetectionStatus
	Targets           []ReShadeTarget
	UnsupportedReason *string
}

type ManagedReShadeRequest = reshade.Request
type ManagedReShadePreview = reshade.Preview
type ManagedReShadeApplyResult = reshade.ApplyResult
type ManagedReShadeRecoveryState = reshade.RecoveryState
type ManagedReShadeTarget = reshade.ManagedTarget
type ManagedReShadeDiscoveryResult = reshade.DiscoveryResult
