package services

import (
	"github.com/phergul/mod-manager/internal/installconfig"
	"github.com/phergul/mod-manager/internal/services/dto"
)

func toDTOStrategyType(strategyType installconfig.StrategyType) dto.StrategyType {
	return dto.StrategyType(strategyType)
}

func toInstallStrategyType(strategyType dto.StrategyType) installconfig.StrategyType {
	return installconfig.StrategyType(strategyType)
}

func toDTOStrategyVisibility(visibility installconfig.StrategyVisibility) dto.StrategyVisibility {
	return dto.StrategyVisibility(visibility)
}

func toDTOStrategyDescriptor(strategy installconfig.StrategyDescriptor) dto.StrategyDescriptor {
	return dto.StrategyDescriptor{
		Type:               toDTOStrategyType(strategy.Type),
		Label:              strategy.Label,
		Description:        strategy.Description,
		Visibility:         toDTOStrategyVisibility(strategy.Visibility),
		RequiresTargetPath: strategy.RequiresTargetPath,
	}
}

func toDTOStrategyDescriptors(strategies []installconfig.StrategyDescriptor) []dto.StrategyDescriptor {
	result := make([]dto.StrategyDescriptor, 0, len(strategies))
	for _, strategy := range strategies {
		result = append(result, toDTOStrategyDescriptor(strategy))
	}
	return result
}

func toDTOPreview(preview installconfig.Preview) dto.Preview {
	return dto.Preview{
		StrategyType:        toDTOStrategyType(preview.StrategyType),
		TargetBase:          preview.TargetBase,
		TargetRelativePath:  preview.TargetRelativePath,
		TargetDisplayPath:   preview.TargetDisplayPath,
		TotalFileCount:      preview.TotalFileCount,
		TotalDirectoryCount: preview.TotalDirectoryCount,
		TargetFilePaths:     preview.TargetFilePaths,
		IsCapped:            preview.IsCapped,
		Cap:                 preview.Cap,
		Warnings:            preview.Warnings,
	}
}
