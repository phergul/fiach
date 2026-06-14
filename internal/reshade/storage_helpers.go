package reshade

import (
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func dbInputFromRow(row dbtypes.ReShadeTarget) dbtypes.SaveReShadeTargetInput {
	return dbtypes.SaveReShadeTargetInput{
		GameID:                 row.GameID,
		TargetRelativePath:     row.TargetRelativePath,
		ExecutableRelativePath: row.ExecutableRelativePath,
		RenderingAPI:           row.RenderingAPI,
		ProxyFilename:          row.ProxyFilename,
		Architecture:           row.Architecture,
		BuildVariant:           row.BuildVariant,
		RuntimeVersion:         row.RuntimeVersion,
		InstallerTag:           row.InstallerTag,
		InstallerAssetName:     row.InstallerAssetName,
		InstallerURL:           row.InstallerURL,
		InstallerDigest:        row.InstallerDigest,
		InstallerSize:          row.InstallerSize,
		ManagementOrigin:       row.ManagementOrigin,
		Status:                 row.Status,
		ManifestJSON:           row.ManifestJSON,
		LastVerifiedAt:         row.LastVerifiedAt,
	}
}
