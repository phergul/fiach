package mappers

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

const (
	modMetadataShortFieldLimit  = 128
	modMetadataSourceURLLimit   = 2048
	modMetadataDescriptionLimit = 4000
	modMetadataNotesLimit       = 8000
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

func ToDTOTag(tag dbtypes.Tag) dto.Tag {
	return dto.Tag{
		ID:     tag.ID,
		GameID: tag.GameID,
		Name:   tag.Name,
		Color:  dto.TagColor(tag.Color),
	}
}

func ToDTOTags(tags []dbtypes.Tag) []dto.Tag {
	result := make([]dto.Tag, 0, len(tags))
	for _, tag := range tags {
		result = append(result, ToDTOTag(tag))
	}
	return result
}

func ToStorageCreateTagInputs(inputs []dto.CreateTagInput) []dbtypes.CreateTagInput {
	result := make([]dbtypes.CreateTagInput, 0, len(inputs))
	for _, input := range inputs {
		result = append(result, dbtypes.CreateTagInput{
			Name:  input.Name,
			Color: dbtypes.TagColor(input.Color),
		})
	}
	return result
}

func ToDTOModWithMetadata(mod dbtypes.Mod, metadata dbtypes.ModMetadata) dto.Mod {
	result := ToDTOMod(mod)
	dtoMetadata := ToDTOModMetadata(metadata)
	result.Metadata = &dtoMetadata
	return result
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

func ToStorageUpdateModMetadataInput(input dto.UpdateModMetadataInput) (dbtypes.UpdateModMetadataInput, error) {
	if input.ModID <= 0 {
		return dbtypes.UpdateModMetadataInput{}, errors.New("mod ID must be positive")
	}

	version, err := toStorageModMetadataFieldUpdate("version", input.Version, modMetadataShortFieldLimit, false)
	if err != nil {
		return dbtypes.UpdateModMetadataInput{}, err
	}
	author, err := toStorageModMetadataFieldUpdate("author", input.Author, modMetadataShortFieldLimit, false)
	if err != nil {
		return dbtypes.UpdateModMetadataInput{}, err
	}
	description, err := toStorageModMetadataFieldUpdate("description", input.Description, modMetadataDescriptionLimit, false)
	if err != nil {
		return dbtypes.UpdateModMetadataInput{}, err
	}
	sourceURL, err := toStorageModMetadataFieldUpdate("source URL", input.SourceURL, modMetadataSourceURLLimit, true)
	if err != nil {
		return dbtypes.UpdateModMetadataInput{}, err
	}
	notes, err := cleanModMetadataText("notes", input.Notes, modMetadataNotesLimit)
	if err != nil {
		return dbtypes.UpdateModMetadataInput{}, err
	}

	return dbtypes.UpdateModMetadataInput{
		ModID:       input.ModID,
		Version:     version,
		Author:      author,
		Description: description,
		SourceURL:   sourceURL,
		Notes:       notes,
	}, nil
}

func toStorageModMetadataFieldUpdate(label string, input dto.ModMetadataFieldUpdate, limit int, validateURL bool) (dbtypes.ModMetadataFieldUpdate, error) {
	switch input.Mode {
	case dto.ModMetadataFieldUpdateModeReset:
		return dbtypes.ModMetadataFieldUpdate{}, nil
	case dto.ModMetadataFieldUpdateModeClear:
		return dbtypes.ModMetadataFieldUpdate{UserSet: true}, nil
	case dto.ModMetadataFieldUpdateModeUser:
		value, err := cleanModMetadataText(label, input.Value, limit)
		if err != nil {
			return dbtypes.ModMetadataFieldUpdate{}, err
		}
		if validateURL && value != nil {
			if err := validateModMetadataSourceURL(*value); err != nil {
				return dbtypes.ModMetadataFieldUpdate{}, err
			}
		}

		return dbtypes.ModMetadataFieldUpdate{UserSet: true, Value: value}, nil
	default:
		return dbtypes.ModMetadataFieldUpdate{}, fmt.Errorf("%s update mode is required", label)
	}
}

func cleanModMetadataText(label string, value *string, limit int) (*string, error) {
	if value == nil {
		return nil, nil
	}

	cleaned := strings.TrimSpace(*value)
	if cleaned == "" {
		return nil, nil
	}
	if len(cleaned) > limit {
		return nil, fmt.Errorf("%s must be %d characters or fewer", label, limit)
	}
	for _, r := range cleaned {
		if unicode.IsControl(r) {
			return nil, fmt.Errorf("%s contains unsupported control characters", label)
		}
	}

	return &cleaned, nil
}

func validateModMetadataSourceURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("source URL is invalid: %w", err)
	}
	if !parsed.IsAbs() || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("source URL must be an absolute http or https URL")
	}

	return nil
}
