package injection

import (
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

type APIFamily string

const (
	APIFamilyDirectX APIFamily = "directx"
	APIFamilyVulkan  APIFamily = "vulkan"
)

type Owner string

const (
	OwnerReShade    Owner = "reshade"
	OwnerOptiScaler Owner = "optiscaler"
)

type Status string

const (
	StatusManaged          Status = "managed"
	StatusDrifted          Status = "drifted"
	StatusRecoveryRequired Status = "recovery_required"
)

type ChainTarget struct {
	GameID                 int64
	TargetRelativePath     string
	ExecutableRelativePath string
	APIFamily              APIFamily
	PrimaryOwner           Owner
	PrimaryProxyFilename   string
	Status                 Status
	OptiScaler             *OptiScalerState
	ReShade                *ReShadeState
}

type OptiScalerState struct {
	ProxyFilename string
	Target        dbtypes.OptiScalerTarget
}

type ReShadeState struct {
	PreferredProxyFilename string
	ActiveRuntimeFilename  string
	Target                 dbtypes.ReShadeTarget
}
