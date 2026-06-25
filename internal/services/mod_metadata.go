package services

import (
	"context"
	"log/slog"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
)

func (s *ModService) GetModMetadata(ctx context.Context, modID int64) (metadata dto.ModMetadata, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationGetModMetadata, "Mod metadata read started",
		slog.Int64("mod_id", modID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Mod metadata read failed", err, modUserError)
		}
	}()

	storedMetadata, found, err := s.store.GetModMetadata(ctx, modID)
	if err != nil {
		return dto.ModMetadata{}, err
	}
	if !found {
		return dto.ModMetadata{}, apperror.New("Mod was not found.")
	}

	metadata = mappers.ToDTOModMetadata(storedMetadata)
	diag.complete("Mod metadata read completed")

	return metadata, nil
}

func (s *ModService) UpdateModMetadata(ctx context.Context, input dto.UpdateModMetadataInput) (metadata dto.ModMetadata, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationUpdateModMetadata, "Mod metadata update started",
		slog.Int64("mod_id", input.ModID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Mod metadata update failed", err, modUserError)
		}
	}()

	if existingMod, found, lookupErr := s.store.GetMod(ctx, input.ModID); lookupErr == nil && found {
		diag.attrs = append(diag.attrs,
			slog.String("mod_name", existingMod.Name),
			slog.Int64("game_id", existingMod.GameID),
		)
	}

	storageInput, err := mappers.ToStorageUpdateModMetadataInput(input)
	if err != nil {
		return dto.ModMetadata{}, err
	}

	storedMetadata, err := s.store.UpdateModMetadata(ctx, storageInput)
	if err != nil {
		return dto.ModMetadata{}, err
	}

	metadata = mappers.ToDTOModMetadata(storedMetadata)
	diag.complete("Mod metadata update completed")

	return metadata, nil
}
