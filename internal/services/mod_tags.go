package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/phergul/fiach/internal/diagnostics"
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
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationRenameTag, "Tag rename started",
		slog.Int64("tag_id", tagID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Tag rename failed", err, modUserError)
		}
	}()

	storedTag, err := s.store.RenameTag(ctx, tagID, name, dbtypes.TagColor(color))
	if err != nil {
		return dto.Tag{}, err
	}

	tag = mappers.ToDTOTag(storedTag)
	diag.complete("Tag rename completed",
		slog.Int64("game_id", storedTag.GameID),
		slog.String("tag_name", storedTag.Name),
	)

	return tag, nil
}

func (s *ModService) UpdateModDetails(ctx context.Context, input dto.UpdateModDetailsInput) (mod dto.Mod, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationUpdateModDetails, "Mod details update started",
		slog.Int64("mod_id", input.ModID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Mod details update failed", err, modUserError)
		}
	}()

	if existingMod, found, lookupErr := s.store.GetMod(ctx, input.ModID); lookupErr == nil && found {
		diag.attrs = append(diag.attrs,
			slog.String("mod_name", existingMod.Name),
			slog.Int64("game_id", existingMod.GameID),
		)
	}

	metadataInput := input.Metadata
	metadataInput.ModID = input.ModID
	storageMetadata, err := mappers.ToStorageUpdateModMetadataInput(metadataInput)
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
	diag.complete("Mod details update completed")

	return result, nil
}
