package mappers

import (
	"github.com/phergul/fiach/internal/loadorder"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func ToDTOModProfile(profile dbtypes.ModProfile) dto.ModProfile {
	return dto.ModProfile(profile)
}

func ToDTOModProfiles(profiles []dbtypes.ModProfile) []dto.ModProfile {
	result := make([]dto.ModProfile, 0, len(profiles))
	for _, profile := range profiles {
		result = append(result, ToDTOModProfile(profile))
	}
	return result
}

func ToDTOProfileMod(profileMod dbtypes.ProfileMod) dto.ProfileMod {
	return dto.ProfileMod{
		ProfileID:        profileMod.ProfileID,
		ModID:            profileMod.ModID,
		Name:             profileMod.Name,
		SourcePath:       profileMod.SourcePath,
		ModUpdatedAt:     profileMod.ModUpdatedAt,
		Enabled:          profileMod.Enabled,
		LoadOrder:        profileMod.LoadOrder,
		DisplayLoadOrder: loadorder.DisplayIndex(profileMod.LoadOrder),
		CreatedAt:        profileMod.CreatedAt,
		UpdatedAt:        profileMod.UpdatedAt,
	}
}

func ToDTOProfileMods(profileMods []dbtypes.ProfileMod) []dto.ProfileMod {
	result := make([]dto.ProfileMod, 0, len(profileMods))
	for _, profileMod := range profileMods {
		result = append(result, ToDTOProfileMod(profileMod))
	}
	return result
}
