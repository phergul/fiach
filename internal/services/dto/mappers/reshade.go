package mappers

import (
	"github.com/phergul/fiach/internal/reshade"
	"github.com/phergul/fiach/internal/services/dto"
)

func ToDTOReShadeTargets(targets []reshade.Target) []dto.ReShadeTarget {
	result := make([]dto.ReShadeTarget, 0, len(targets))
	for _, target := range targets {
		result = append(result, dto.ReShadeTarget{
			Path:        target.Path,
			Executables: append([]string(nil), target.Executables...),
		})
	}

	return result
}
