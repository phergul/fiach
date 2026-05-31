package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

const (
	modMetadataShortFieldLimit  = 128
	modMetadataSourceURLLimit   = 2048
	modMetadataDescriptionLimit = 4000
	modMetadataNotesLimit       = 8000
)

func (s *ModService) GetModMetadata(ctx context.Context, modID int64) (metadata dto.ModMetadata, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get mod metadata: %w", err)
		}
	}()

	storedMetadata, found, err := s.store.GetModMetadata(ctx, modID)
	if err != nil {
		return dto.ModMetadata{}, err
	}
	if !found {
		return dto.ModMetadata{}, fmt.Errorf("mod %d was not found", modID)
	}

	return mappers.ToDTOModMetadata(storedMetadata), nil
}

func (s *ModService) UpdateModMetadata(ctx context.Context, input dto.UpdateModMetadataInput) (metadata dto.ModMetadata, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update mod metadata: %w", err)
		}
	}()

	storageInput, err := toStorageUpdateModMetadataInput(input)
	if err != nil {
		return dto.ModMetadata{}, err
	}

	storedMetadata, err := s.store.UpdateModMetadata(ctx, storageInput)
	if err != nil {
		return dto.ModMetadata{}, err
	}

	return mappers.ToDTOModMetadata(storedMetadata), nil
}

func toStorageUpdateModMetadataInput(input dto.UpdateModMetadataInput) (dbtypes.UpdateModMetadataInput, error) {
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
