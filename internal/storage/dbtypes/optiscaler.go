package dbtypes

type OptiScalerTarget struct {
	ID                     int64   `db:"id"`
	GameID                 int64   `db:"game_id"`
	TargetRelativePath     string  `db:"target_relative_path"`
	ExecutableRelativePath string  `db:"executable_relative_path"`
	GraphicsAPI            string  `db:"graphics_api"`
	ProxyFilename          string  `db:"proxy_filename"`
	DXGISpoofing           bool    `db:"dxgi_spoofing"`
	ProcessFilter          *string `db:"process_filter"`
	ReleaseTag             string  `db:"release_tag"`
	ReleaseVersion         string  `db:"release_version"`
	ReleaseAssetName       string  `db:"release_asset_name"`
	ReleaseDigest          string  `db:"release_digest"`
	ManagementOrigin       string  `db:"management_origin"`
	Status                 string  `db:"status"`
	ManifestJSON           string  `db:"manifest_json"`
	WarningVersion         string  `db:"warning_version"`
	WarningAcknowledgedAt  *string `db:"warning_acknowledged_at"`
	CreatedAt              string  `db:"created_at"`
	UpdatedAt              string  `db:"updated_at"`
	LastVerifiedAt         *string `db:"last_verified_at"`
}

type SaveOptiScalerTargetInput struct {
	GameID                 int64
	TargetRelativePath     string
	ExecutableRelativePath string
	GraphicsAPI            string
	ProxyFilename          string
	DXGISpoofing           bool
	ProcessFilter          *string
	ReleaseTag             string
	ReleaseVersion         string
	ReleaseAssetName       string
	ReleaseDigest          string
	ManagementOrigin       string
	Status                 string
	ManifestJSON           string
	WarningVersion         string
	WarningAcknowledgedAt  *string
	LastVerifiedAt         *string
}
