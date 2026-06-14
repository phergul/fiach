package mappers

import (
	"encoding/json"

	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func ToDTOOptiScalerTarget(target dbtypes.OptiScalerTarget) dto.OptiScalerTarget {
	var manifest struct {
		Config struct {
			LoadReShade bool `json:"loadReShade"`
		} `json:"config"`
	}
	_ = json.Unmarshal([]byte(target.ManifestJSON), &manifest)
	return dto.OptiScalerTarget{
		ID: target.ID, GameID: target.GameID,
		TargetRelativePath:       target.TargetRelativePath,
		ExecutableRelativePath:   target.ExecutableRelativePath,
		GraphicsAPI:              target.GraphicsAPI,
		ProxyFilename:            target.ProxyFilename,
		DXGISpoofing:             target.DXGISpoofing,
		ProcessFilter:            target.ProcessFilter,
		EnableReShadeCoexistence: manifest.Config.LoadReShade,
		ReleaseTag:               target.ReleaseTag,
		ReleaseVersion:           target.ReleaseVersion,
		ReleaseAssetName:         target.ReleaseAssetName,
		ReleaseDigest:            target.ReleaseDigest,
		ManagementOrigin:         target.ManagementOrigin,
		Status:                   target.Status,
		WarningVersion:           target.WarningVersion,
		WarningAcknowledgedAt:    target.WarningAcknowledgedAt,
		CreatedAt:                target.CreatedAt,
		UpdatedAt:                target.UpdatedAt,
		LastVerifiedAt:           target.LastVerifiedAt,
	}
}
