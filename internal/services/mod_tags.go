package services

import (
	"context"
	"fmt"

	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func (s *ModService) ListGameTags(ctx context.Context, gameID int64) (tags []dto.Tag, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list game tags: %w", err)
		}
	}()

	storedTags, err := s.store.ListGameTags(ctx, gameID)
	if err != nil {
		return nil, err
	}
	return mappers.ToDTOTags(storedTags), nil
}

func (s *ModService) RenameTag(ctx context.Context, tagID int64, name string, color dto.TagColor) (tag dto.Tag, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("rename tag: %w", err)
		}
	}()

	storedTag, err := s.store.RenameTag(ctx, tagID, name, dbtypes.TagColor(color))
	if err != nil {
		return dto.Tag{}, err
	}
	return mappers.ToDTOTag(storedTag), nil
}

func (s *ModService) UpdateModDetails(ctx context.Context, input dto.UpdateModDetailsInput) (mod dto.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update mod details: %w", err)
		}
	}()

	metadataInput := input.Metadata
	metadataInput.ModID = input.ModID
	storageMetadata, err := toStorageUpdateModMetadataInput(metadataInput)
	if err != nil {
		return dto.Mod{}, err
	}

	storedMod, storedMetadata, storedTags, err := s.store.UpdateModDetails(ctx, dbtypes.UpdateModDetailsInput{
		ModID:    input.ModID,
		Name:     input.Name,
		Metadata: storageMetadata,
		TagIDs:   input.TagIDs,
		NewTags:  mappers.ToStorageCreateTagInputs(input.NewTags),
	})
	if err != nil {
		return dto.Mod{}, err
	}

	result := mappers.ToDTOModWithMetadata(storedMod, storedMetadata)
	result.Tags = mappers.ToDTOTags(storedTags)
	return result, nil
}

