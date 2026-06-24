package mappers

import (
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
