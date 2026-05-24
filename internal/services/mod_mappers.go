package services

import (
	"github.com/phergul/mod-manager/internal/services/dto"
	"github.com/phergul/mod-manager/internal/storage/dbtypes"
)

func toDTOModSourceType(sourceType dbtypes.ModSourceType) dto.ModSourceType {
	return dto.ModSourceType(sourceType)
}

func toDBModSourceType(sourceType dto.ModSourceType) dbtypes.ModSourceType {
	return dbtypes.ModSourceType(sourceType)
}

func toDTOMod(mod dbtypes.Mod) dto.Mod {
	return dto.Mod{
		ID:                 mod.ID,
		GameID:             mod.GameID,
		Name:               mod.Name,
		SourceType:         toDTOModSourceType(mod.SourceType),
		SourcePath:         mod.SourcePath,
		OriginalSourcePath: mod.OriginalSourcePath,
		OriginalSourceName: mod.OriginalSourceName,
		CreatedAt:          mod.CreatedAt,
		UpdatedAt:          mod.UpdatedAt,
	}
}

func toDTOMods(mods []dbtypes.Mod) []dto.Mod {
	result := make([]dto.Mod, 0, len(mods))
	for _, mod := range mods {
		result = append(result, toDTOMod(mod))
	}
	return result
}

func toDTOModInstallConfig(config dbtypes.ModInstallConfig) dto.ModInstallConfig {
	return dto.ModInstallConfig(config)
}
