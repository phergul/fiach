package dbtypes

type ReShadeTarget struct {
	ID                     int64   `db:"id"`
	GameID                 int64   `db:"game_id"`
	TargetRelativePath     string  `db:"target_relative_path"`
	ExecutableRelativePath string  `db:"executable_relative_path"`
	RenderingAPI           string  `db:"rendering_api"`
	ProxyFilename          string  `db:"proxy_filename"`
	ActiveRuntimeFilename  string  `db:"active_runtime_filename"`
	Architecture           string  `db:"architecture"`
	BuildVariant           string  `db:"build_variant"`
	RuntimeVersion         string  `db:"runtime_version"`
	InstallerTag           *string `db:"installer_tag"`
	InstallerAssetName     *string `db:"installer_asset_name"`
	InstallerURL           *string `db:"installer_url"`
	InstallerDigest        *string `db:"installer_digest"`
	InstallerSize          *int64  `db:"installer_size"`
	ManagementOrigin       string  `db:"management_origin"`
	Status                 string  `db:"status"`
	ManifestJSON           string  `db:"manifest_json"`
	CreatedAt              string  `db:"created_at"`
	UpdatedAt              string  `db:"updated_at"`
	LastVerifiedAt         *string `db:"last_verified_at"`
}

type SaveReShadeTargetInput struct {
	GameID                 int64
	TargetRelativePath     string
	ExecutableRelativePath string
	RenderingAPI           string
	ProxyFilename          string
	ActiveRuntimeFilename  string
	Architecture           string
	BuildVariant           string
	RuntimeVersion         string
	InstallerTag           *string
	InstallerAssetName     *string
	InstallerURL           *string
	InstallerDigest        *string
	InstallerSize          *int64
	ManagementOrigin       string
	Status                 string
	ManifestJSON           string
	LastVerifiedAt         *string
}
