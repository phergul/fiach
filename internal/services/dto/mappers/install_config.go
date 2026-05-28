package mappers

import (
	"github.com/phergul/mod-manager/internal/installconfig"
	"github.com/phergul/mod-manager/internal/services/dto"
)

func ToDTOStrategyType(strategyType installconfig.StrategyType) dto.StrategyType {
	return dto.StrategyType(strategyType)
}

func ToInstallStrategyType(strategyType dto.StrategyType) installconfig.StrategyType {
	return installconfig.StrategyType(strategyType)
}

func ToDTOStrategyVisibility(visibility installconfig.StrategyVisibility) dto.StrategyVisibility {
	return dto.StrategyVisibility(visibility)
}

func ToDTOStrategyDescriptor(strategy installconfig.StrategyDescriptor) dto.StrategyDescriptor {
	return dto.StrategyDescriptor{
		Type:               ToDTOStrategyType(strategy.Type),
		Label:              strategy.Label,
		Description:        strategy.Description,
		Visibility:         ToDTOStrategyVisibility(strategy.Visibility),
		RequiresTargetPath: strategy.RequiresTargetPath,
	}
}

func ToDTOStrategyDescriptors(strategies []installconfig.StrategyDescriptor) []dto.StrategyDescriptor {
	result := make([]dto.StrategyDescriptor, 0, len(strategies))
	for _, strategy := range strategies {
		result = append(result, ToDTOStrategyDescriptor(strategy))
	}
	return result
}

func ToDTOPreview(preview installconfig.Preview) dto.Preview {
	return dto.Preview{
		StrategyType:        ToDTOStrategyType(preview.StrategyType),
		TargetBase:          preview.TargetBase,
		TargetRelativePath:  preview.TargetRelativePath,
		TargetDisplayPath:   preview.TargetDisplayPath,
		TotalFileCount:      preview.TotalFileCount,
		TotalDirectoryCount: preview.TotalDirectoryCount,
		TotalSizeBytes:      preview.TotalSizeBytes,
		TargetFilePaths:     preview.TargetFilePaths,
		IsCapped:            preview.IsCapped,
		Cap:                 preview.Cap,
		Warnings:            preview.Warnings,
	}
}
