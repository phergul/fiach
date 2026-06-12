package mappers

import (
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func ToModPackageSnapshot(mod dbtypes.Mod, metadata dbtypes.ModMetadata) dto.ModPackageSnapshot {
	return dto.ModPackageSnapshot{
		SourceType:         ToDTOModSourceType(mod.SourceType),
		OriginalSourcePath: mod.OriginalSourcePath,
		OriginalSourceName: mod.OriginalSourceName,
		FileCount:          mod.FileCount,
		DirectoryCount:     mod.DirectoryCount,
		TotalSizeBytes:     mod.TotalSizeBytes,
		DetectedMetadata: dto.ModDetectedMetadataSnapshot{
			Version:     metadata.DetectedVersion,
			Author:      metadata.DetectedAuthor,
			Description: metadata.DetectedDescription,
			SourceURL:   metadata.DetectedSourceURL,
		},
	}
}
