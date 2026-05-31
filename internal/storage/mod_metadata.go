package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

const modMetadataSelectColumns = `
	mod_id,
	detected_version,
	user_version,
	version_user_set,
	detected_author,
	user_author,
	author_user_set,
	detected_description,
	user_description,
	description_user_set,
	detected_source_url,
	user_source_url,
	source_url_user_set,
	notes,
	created_at,
	updated_at
`

func (s *Store) GetModMetadata(ctx context.Context, modID int64) (metadata dbtypes.ModMetadata, found bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select mod metadata row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.ModMetadata{}, false, errors.New("store is not open")
	}
	if modID <= 0 {
		return dbtypes.ModMetadata{}, false, errors.New("mod ID must be positive")
	}

	if _, found, err := s.GetMod(ctx, modID); err != nil {
		return dbtypes.ModMetadata{}, false, err
	} else if !found {
		return dbtypes.ModMetadata{}, false, nil
	}

	metadata, found, err = getModMetadata(ctx, s.db, modID)
	if err != nil {
		return dbtypes.ModMetadata{}, false, err
	}
	if !found {
		return dbtypes.ModMetadata{ModID: modID}, true, nil
	}

	return metadata, true, nil
}

func (s *Store) UpdateModMetadata(ctx context.Context, input dbtypes.UpdateModMetadataInput) (metadata dbtypes.ModMetadata, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update mod metadata row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.ModMetadata{}, errors.New("store is not open")
	}
	if input.ModID <= 0 {
		return dbtypes.ModMetadata{}, errors.New("mod ID must be positive")
	}

	if _, found, err := s.GetMod(ctx, input.ModID); err != nil {
		return dbtypes.ModMetadata{}, err
	} else if !found {
		return dbtypes.ModMetadata{}, sql.ErrNoRows
	}

	if err := upsertModMetadataUserValues(ctx, s.db, input); err != nil {
		return dbtypes.ModMetadata{}, err
	}

	metadata, found, err := getModMetadata(ctx, s.db, input.ModID)
	if err != nil {
		return dbtypes.ModMetadata{}, err
	}
	if !found {
		return dbtypes.ModMetadata{}, sql.ErrNoRows
	}

	return metadata, nil
}

func (s *Store) UpdateModDetectedMetadata(ctx context.Context, modID int64, input dbtypes.ModMetadataDetectedInput) (metadata dbtypes.ModMetadata, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update detected mod metadata row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.ModMetadata{}, errors.New("store is not open")
	}
	if modID <= 0 {
		return dbtypes.ModMetadata{}, errors.New("mod ID must be positive")
	}

	if _, found, err := s.GetMod(ctx, modID); err != nil {
		return dbtypes.ModMetadata{}, err
	} else if !found {
		return dbtypes.ModMetadata{}, sql.ErrNoRows
	}

	if err := upsertDetectedModMetadata(ctx, s.db, modID, input); err != nil {
		return dbtypes.ModMetadata{}, err
	}

	metadata, found, err := getModMetadata(ctx, s.db, modID)
	if err != nil {
		return dbtypes.ModMetadata{}, err
	}
	if !found {
		return dbtypes.ModMetadata{}, sql.ErrNoRows
	}

	return metadata, nil
}

func getModMetadata(ctx context.Context, db modGetter, modID int64) (dbtypes.ModMetadata, bool, error) {
	var metadata dbtypes.ModMetadata
	err := db.GetContext(ctx, &metadata, `
		SELECT `+modMetadataSelectColumns+`
		FROM mod_metadata
		WHERE mod_id = ?
	`, modID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dbtypes.ModMetadata{}, false, nil
		}

		return dbtypes.ModMetadata{}, false, err
	}

	return metadata, true, nil
}

func upsertDetectedModMetadata(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, modID int64, input dbtypes.ModMetadataDetectedInput) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO mod_metadata (
			mod_id,
			detected_version,
			detected_author,
			detected_description,
			detected_source_url
		)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(mod_id) DO UPDATE SET
			detected_version = excluded.detected_version,
			detected_author = excluded.detected_author,
			detected_description = excluded.detected_description,
			detected_source_url = excluded.detected_source_url,
			updated_at = CURRENT_TIMESTAMP
	`, modID, nullableText(cleanOptionalString(input.Version)), nullableText(cleanOptionalString(input.Author)), nullableText(cleanOptionalString(input.Description)), nullableText(cleanOptionalString(input.SourceURL)))
	if err != nil {
		return err
	}

	return nil
}

func upsertModMetadataUserValues(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, input dbtypes.UpdateModMetadataInput) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO mod_metadata (
			mod_id,
			user_version,
			version_user_set,
			user_author,
			author_user_set,
			user_description,
			description_user_set,
			user_source_url,
			source_url_user_set,
			notes
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(mod_id) DO UPDATE SET
			user_version = excluded.user_version,
			version_user_set = excluded.version_user_set,
			user_author = excluded.user_author,
			author_user_set = excluded.author_user_set,
			user_description = excluded.user_description,
			description_user_set = excluded.description_user_set,
			user_source_url = excluded.user_source_url,
			source_url_user_set = excluded.source_url_user_set,
			notes = excluded.notes,
			updated_at = CURRENT_TIMESTAMP
	`, input.ModID,
		nullableText(cleanOptionalString(input.Version.Value)), input.Version.UserSet,
		nullableText(cleanOptionalString(input.Author.Value)), input.Author.UserSet,
		nullableText(cleanOptionalString(input.Description.Value)), input.Description.UserSet,
		nullableText(cleanOptionalString(input.SourceURL.Value)), input.SourceURL.UserSet,
		nullableText(cleanOptionalString(input.Notes)),
	)
	if err != nil {
		return err
	}

	return nil
}
