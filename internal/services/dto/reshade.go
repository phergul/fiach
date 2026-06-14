package dto

import "github.com/phergul/fiach/internal/optiscaler"

type ReShadeDetectionStatus string

const (
	ReShadeDetectionStatusInstalled    ReShadeDetectionStatus = "installed"
	ReShadeDetectionStatusNotInstalled ReShadeDetectionStatus = "not_installed"
	ReShadeDetectionStatusUnsupported  ReShadeDetectionStatus = "unsupported"
)

type ReShadeTarget struct {
	Path        string
	Executables []string
}

type ReShadeDetectionResult struct {
	Status            ReShadeDetectionStatus
	Targets           []ReShadeTarget
	UnsupportedReason *string
}

type ReShadeInstallerLaunchResult struct {
	Version string
}

type ReShadeInstallerPreflightDisposition string

const (
	ReShadeInstallerPreflightOrdinary    ReShadeInstallerPreflightDisposition = "ordinary"
	ReShadeInstallerPreflightCoordinated ReShadeInstallerPreflightDisposition = "coordinated"
	ReShadeInstallerPreflightBlocked     ReShadeInstallerPreflightDisposition = "blocked"
)

type ReShadeManagedTarget struct {
	TargetRelativePath     string
	ExecutableRelativePath string
	ProxyFilename          string
}

type ReShadeInstallerPreflight struct {
	Disposition ReShadeInstallerPreflightDisposition
	Variant     optiscaler.ReShadeInstallerVariant
	Targets     []ReShadeManagedTarget
	Message     string
}
