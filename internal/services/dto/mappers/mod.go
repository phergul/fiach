package mappers

import (
	"github.com/phergul/mod-manager/internal/services/dto"
	"github.com/phergul/mod-manager/internal/storage/dbtypes"
)

func ToDTOModSourceType(sourceType dbtypes.ModSourceType) dto.ModSourceType {
	return dto.ModSourceType(sourceType)
}

func ToDBModSourceType(sourceType dto.ModSourceType) dbtypes.ModSourceType {
	return dbtypes.ModSourceType(sourceType)
}

func ToDTOMod(mod dbtypes.Mod) dto.Mod {
	return dto.Mod{
		ID:                 mod.ID,
		GameID:             mod.GameID,
		Name:               mod.Name,
		SourceType:         ToDTOModSourceType(mod.SourceType),
		SourcePath:         mod.SourcePath,
		OriginalSourcePath: mod.OriginalSourcePath,
		OriginalSourceName: mod.OriginalSourceName,
		CreatedAt:          mod.CreatedAt,
		UpdatedAt:          mod.UpdatedAt,
	}
}

func ToDTOMods(mods []dbtypes.Mod) []dto.Mod {
	result := make([]dto.Mod, 0, len(mods))
	for _, mod := range mods {
		result = append(result, ToDTOMod(mod))
	}
	return result
}

func ToDTOModInstallConfig(config dbtypes.ModInstallConfig) dto.ModInstallConfig {
	return dto.ModInstallConfig(config)
}
