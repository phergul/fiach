package mappers

import (
	"github.com/phergul/fiach/internal/injection"
	"github.com/phergul/fiach/internal/reshade"
	"github.com/phergul/fiach/internal/services/dto"
)

func ToDTOReShadeTargets(targets []reshade.Target) []dto.DetectedReShadeTarget {
	result := make([]dto.DetectedReShadeTarget, 0, len(targets))
	for _, target := range targets {
		result = append(result, dto.DetectedReShadeTarget{
			Path:             target.Path,
			Executables:      append([]string(nil), target.Executables...),
			ManagementStatus: reshade.ManagementStatusUnmanaged,
		})
	}

	return result
}

func ToDTOReShadeChainTarget(target injection.ChainTarget) dto.ReShadeChainTarget {
	result := dto.ReShadeChainTarget{
		GameID:                 target.GameID,
		TargetRelativePath:     target.TargetRelativePath,
		ExecutableRelativePath: target.ExecutableRelativePath,
		APIFamily:              target.APIFamily,
		PrimaryOwner:           target.PrimaryOwner,
		PrimaryProxyFilename:   target.PrimaryProxyFilename,
		Status:                 target.Status,
	}
	if target.OptiScaler != nil {
		result.OptiScaler = &dto.ReShadeOptiScalerChainState{
			ProxyFilename: target.OptiScaler.ProxyFilename,
			Status:        target.OptiScaler.Target.Status,
		}
	}
	if target.ReShade != nil {
		result.ReShade = &dto.ReShadeChainState{
			PreferredProxyFilename: target.ReShade.PreferredProxyFilename,
			ActiveRuntimeFilename:  target.ReShade.ActiveRuntimeFilename,
			Status:                 target.ReShade.Target.Status,
		}
	}

	return result
}
