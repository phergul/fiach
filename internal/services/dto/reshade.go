package dto

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
