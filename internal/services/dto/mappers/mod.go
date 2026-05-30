package mappers

import (
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage/dbtypes"
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
		FileCount:          mod.FileCount,
		DirectoryCount:     mod.DirectoryCount,
		TotalSizeBytes:     mod.TotalSizeBytes,
		MetadataJSON:       mod.MetadataJSON,
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

func ToDTOModDeleteSummary(mod dbtypes.Mod, profileUsageCount int64, isInAppliedProfile bool) dto.ModDeleteSummary {
	return dto.ModDeleteSummary{
		ModID:              mod.ID,
		ModName:            mod.Name,
		ProfileUsageCount:  profileUsageCount,
		IsInAppliedProfile: isInAppliedProfile,
		ManagedSourcePath:  mod.SourcePath,
		OriginalSourceName: mod.OriginalSourceName,
		OriginalSourcePath: mod.OriginalSourcePath,
	}
}

func ToDTOModInstallConfig(config dbtypes.ModInstallConfig) dto.ModInstallConfig {
	return dto.ModInstallConfig(config)
}
