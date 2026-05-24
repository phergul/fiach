package services

import (
	"github.com/phergul/mod-manager/internal/services/dto"
	"github.com/phergul/mod-manager/internal/storage/dbtypes"
)

func toDTOModProfile(profile dbtypes.ModProfile) dto.ModProfile {
	return dto.ModProfile(profile)
}

func toDTOModProfiles(profiles []dbtypes.ModProfile) []dto.ModProfile {
	result := make([]dto.ModProfile, 0, len(profiles))
	for _, profile := range profiles {
		result = append(result, toDTOModProfile(profile))
	}
	return result
}

func toDTOProfileMod(profileMod dbtypes.ProfileMod) dto.ProfileMod {
	return dto.ProfileMod(profileMod)
}

func toDTOProfileMods(profileMods []dbtypes.ProfileMod) []dto.ProfileMod {
	result := make([]dto.ProfileMod, 0, len(profileMods))
	for _, profileMod := range profileMods {
		result = append(result, toDTOProfileMod(profileMod))
	}
	return result
}
