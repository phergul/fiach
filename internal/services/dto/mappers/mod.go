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

func ToDTOModMetadata(metadata dbtypes.ModMetadata) dto.ModMetadata {
	return dto.ModMetadata{
		ModID: metadata.ModID,
		Version: dto.ModMetadataField{
			Detected:  metadata.DetectedVersion,
			User:      metadata.UserVersion,
			UserSet:   metadata.VersionUserSet,
			Effective: effectiveMetadataValue(metadata.DetectedVersion, metadata.UserVersion, metadata.VersionUserSet),
		},
		Author: dto.ModMetadataField{
			Detected:  metadata.DetectedAuthor,
			User:      metadata.UserAuthor,
			UserSet:   metadata.AuthorUserSet,
			Effective: effectiveMetadataValue(metadata.DetectedAuthor, metadata.UserAuthor, metadata.AuthorUserSet),
		},
		Description: dto.ModMetadataField{
			Detected:  metadata.DetectedDescription,
			User:      metadata.UserDescription,
			UserSet:   metadata.DescriptionUserSet,
			Effective: effectiveMetadataValue(metadata.DetectedDescription, metadata.UserDescription, metadata.DescriptionUserSet),
		},
		SourceURL: dto.ModMetadataField{
			Detected:  metadata.DetectedSourceURL,
			User:      metadata.UserSourceURL,
			UserSet:   metadata.SourceURLUserSet,
			Effective: effectiveMetadataValue(metadata.DetectedSourceURL, metadata.UserSourceURL, metadata.SourceURLUserSet),
		},
		Notes:     metadata.Notes,
		CreatedAt: metadata.CreatedAt,
		UpdatedAt: metadata.UpdatedAt,
	}
}

func effectiveMetadataValue(detected *string, user *string, userSet bool) *string {
	if userSet {
		return user
	}

	return detected
}

func ToDTOModInstallConfig(config dbtypes.ModInstallConfig) dto.ModInstallConfig {
	return dto.ModInstallConfig(config)
}
