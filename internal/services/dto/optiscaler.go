package dto

import (
	"github.com/phergul/fiach/internal/optiscaler"
)

type OptiScalerCandidate = optiscaler.Candidate
type OptiScalerRelease = optiscaler.Release
type OptiScalerRequest = optiscaler.Request
type OptiScalerPreview = optiscaler.Preview
type OptiScalerApplyResult = optiscaler.ApplyResult
type OptiScalerRecoveryState = optiscaler.RecoveryState

type OptiScalerTarget struct {
	ID                       int64
	GameID                   int64
	TargetRelativePath       string
	ExecutableRelativePath   string
	GraphicsAPI              string
	ProxyFilename            string
	DXGISpoofing             bool
	ProcessFilter            *string
	EnableReShadeCoexistence bool
	ReleaseTag               string
	ReleaseVersion           string
	ReleaseAssetName         string
	ReleaseDigest            string
	ManagementOrigin         string
	Status                   string
	WarningVersion           string
	WarningAcknowledgedAt    *string
	CreatedAt                string
	UpdatedAt                string
	LastVerifiedAt           *string
}
