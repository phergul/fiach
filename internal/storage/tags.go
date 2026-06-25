package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jmoiron/sqlx"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

const (
	tagNameLimit = 50
	modTagLimit  = 20
)

const tagSelectColumns = `
	id,
	game_id,
	name,
	normalized_name,
	color,
	created_at,
	updated_at
`

const qualifiedTagSelectColumns = `
	t.id AS id,
	t.game_id AS game_id,
	t.name AS name,
	t.normalized_name AS normalized_name,
	t.color AS color,
	t.created_at AS created_at,
	t.updated_at AS updated_at
`

func (s *Store) ListGameTags(ctx context.Context, gameID int64) (tags []dbtypes.Tag, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select game tags: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}
	if gameID <= 0 {
		return nil, errors.New("game ID must be positive")
	}

	err = s.db.SelectContext(ctx, &tags, `
		SELECT `+tagSelectColumns+`
		FROM tags
		WHERE game_id = ?
		ORDER BY normalized_name, id
	`, gameID)
	return tags, err
}

func (s *Store) ListTagsForMods(ctx context.Context, modIDs []int64) (tagsByModID map[int64][]dbtypes.Tag, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select tags for mods: %w", err)
		}
	}()

	tagsByModID = make(map[int64][]dbtypes.Tag, len(modIDs))
	if len(modIDs) == 0 {
		return tagsByModID, nil
	}
	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}

	type modTagRow struct {
		ModID int64 `db:"mod_id"`
		dbtypes.Tag
	}

	query, args, err := sqlx.In(`
		SELECT mt.mod_id, `+qualifiedTagSelectColumns+`
		FROM mod_tags mt
		INNER JOIN tags t ON t.id = mt.tag_id
		WHERE mt.mod_id IN (?)
		ORDER BY t.normalized_name, t.id
	`, modIDs)
	if err != nil {
		return nil, err
	}

	var rows []modTagRow
	if err := s.db.SelectContext(ctx, &rows, s.db.Rebind(query), args...); err != nil {
		return nil, err
	}
	for _, row := range rows {
		tagsByModID[row.ModID] = append(tagsByModID[row.ModID], row.Tag)
	}

	return tagsByModID, nil
}

func (s *Store) SetModTags(ctx context.Context, input dbtypes.SetModTagsInput) (tags []dbtypes.Tag, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update mod tag assignments: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}

	err = withTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		var updateErr error
		tags, updateErr = setModTags(ctx, tx, input)
		return updateErr
	})
	if err != nil {
		return nil, err
	}

	return tags, nil
}

func (s *Store) UpdateModDetails(ctx context.Context, input dbtypes.UpdateModDetailsInput) (mod dbtypes.Mod, metadata dbtypes.ModMetadata, tags []dbtypes.Tag, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update mod details rows: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.Mod{}, dbtypes.ModMetadata{}, nil, errors.New("store is not open")
	}
	if input.ModID <= 0 {
		return dbtypes.Mod{}, dbtypes.ModMetadata{}, nil, errors.New("mod ID must be positive")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return dbtypes.Mod{}, dbtypes.ModMetadata{}, nil, errors.New("mod name is required")
	}

	err = withTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE mods
			SET name = ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, name, input.ModID)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("get updated mod count: %w", err)
		}
		if affected == 0 {
			return sql.ErrNoRows
		}

		metadataInput := input.Metadata
		metadataInput.ModID = input.ModID
		if err := upsertModMetadataUserValues(ctx, tx, metadataInput); err != nil {
			return err
		}
		if _, err := setModTags(ctx, tx, dbtypes.SetModTagsInput{
			ModID:   input.ModID,
			TagIDs:  input.TagIDs,
			NewTags: input.NewTags,
		}); err != nil {
			return err
		}

		mod, err = getModByID(ctx, tx, input.ModID)
		if err != nil {
			return err
		}
		metadata, _, err = getModMetadata(ctx, tx, input.ModID)
		return err
	})
	if err != nil {
		return dbtypes.Mod{}, dbtypes.ModMetadata{}, nil, err
	}

	tagsByModID, err := s.ListTagsForMods(ctx, []int64{input.ModID})
	if err != nil {
		return dbtypes.Mod{}, dbtypes.ModMetadata{}, nil, err
	}
	return mod, metadata, tagsByModID[input.ModID], nil
}

func (s *Store) RenameTag(ctx context.Context, tagID int64, name string, color dbtypes.TagColor) (tag dbtypes.Tag, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("rename game tag: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.Tag{}, errors.New("store is not open")
	}
	if tagID <= 0 {
		return dbtypes.Tag{}, errors.New("tag ID must be positive")
	}

	cleanName, normalizedName, err := validateTagName(name)
	if err != nil {
		return dbtypes.Tag{}, err
	}
	if err := validateTagColor(color); err != nil {
		return dbtypes.Tag{}, err
	}

	err = withTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		current, err := getTagByID(ctx, tx, tagID)
		if err != nil {
			return err
		}

		var existing dbtypes.Tag
		err = tx.GetContext(ctx, &existing, `
			SELECT `+tagSelectColumns+`
			FROM tags
			WHERE game_id = ?
				AND normalized_name = ?
				AND id <> ?
		`, current.GameID, normalizedName, current.ID)
		switch {
		case err == nil:
			if _, err := tx.ExecContext(ctx, `
				INSERT OR IGNORE INTO mod_tags (mod_id, tag_id)
				SELECT mod_id, ?
				FROM mod_tags
				WHERE tag_id = ?
			`, existing.ID, current.ID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, current.ID); err != nil {
				return err
			}
			tag = existing
			return nil
		case !errors.Is(err, sql.ErrNoRows):
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE tags
			SET name = ?,
				normalized_name = ?,
				color = ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, cleanName, normalizedName, color, current.ID); err != nil {
			return mapSQLiteError(err)
		}

		tag, err = getTagByID(ctx, tx, current.ID)
		return err
	})
	if err != nil {
		return dbtypes.Tag{}, err
	}

	return tag, nil
}

func setModTags(ctx context.Context, tx *sqlx.Tx, input dbtypes.SetModTagsInput) ([]dbtypes.Tag, error) {
	if input.ModID <= 0 {
		return nil, errors.New("mod ID must be positive")
	}

	mod, err := getModByID(ctx, tx, input.ModID)
	if err != nil {
		return nil, err
	}

	tagIDs := make([]int64, 0, len(input.TagIDs)+len(input.NewTags))
	seenIDs := make(map[int64]struct{}, cap(tagIDs))
	for _, tagID := range input.TagIDs {
		if tagID <= 0 {
			return nil, errors.New("tag IDs must be positive")
		}
		if _, found := seenIDs[tagID]; found {
			continue
		}

		tag, err := getTagByID(ctx, tx, tagID)
		if err != nil {
			return nil, err
		}
		if tag.GameID != mod.GameID {
			return nil, fmt.Errorf("tag %d does not belong to mod game %d", tagID, mod.GameID)
		}

		seenIDs[tagID] = struct{}{}
		tagIDs = append(tagIDs, tagID)
	}

	seenNames := make(map[string]struct{}, len(input.NewTags))
	for _, newTag := range input.NewTags {
		cleanName, normalizedName, err := validateTagName(newTag.Name)
		if err != nil {
			return nil, err
		}
		if err := validateTagColor(newTag.Color); err != nil {
			return nil, err
		}
		if _, found := seenNames[normalizedName]; found {
			continue
		}
		seenNames[normalizedName] = struct{}{}

		tag, err := upsertTag(ctx, tx, mod.GameID, cleanName, normalizedName, newTag.Color)
		if err != nil {
			return nil, err
		}
		if _, found := seenIDs[tag.ID]; found {
			continue
		}
		seenIDs[tag.ID] = struct{}{}
		tagIDs = append(tagIDs, tag.ID)
	}

	if input.MergeOnly {
		var existingIDs []int64
		if err := tx.SelectContext(ctx, &existingIDs, `SELECT tag_id FROM mod_tags WHERE mod_id = ?`, input.ModID); err != nil {
			return nil, err
		}
		for _, tagID := range existingIDs {
			if _, found := seenIDs[tagID]; found {
				continue
			}
			seenIDs[tagID] = struct{}{}
			tagIDs = append(tagIDs, tagID)
		}
	}
	if len(tagIDs) > modTagLimit {
		return nil, fmt.Errorf("a mod may have at most %d tags", modTagLimit)
	}

	if !input.MergeOnly {
		if _, err := tx.ExecContext(ctx, `DELETE FROM mod_tags WHERE mod_id = ?`, input.ModID); err != nil {
			return nil, err
		}
	}
	for _, tagID := range tagIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO mod_tags (mod_id, tag_id)
			VALUES (?, ?)
		`, input.ModID, tagID); err != nil {
			return nil, err
		}
	}

	var tags []dbtypes.Tag
	if err := tx.SelectContext(ctx, &tags, `
		SELECT `+qualifiedTagSelectColumns+`
		FROM tags t
		INNER JOIN mod_tags mt ON mt.tag_id = t.id
		WHERE mt.mod_id = ?
		ORDER BY t.normalized_name, t.id
	`, input.ModID); err != nil {
		return nil, err
	}
	return tags, nil
}

func upsertTag(ctx context.Context, tx *sqlx.Tx, gameID int64, name string, normalizedName string, color dbtypes.TagColor) (dbtypes.Tag, error) {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO tags (game_id, name, normalized_name, color)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(game_id, normalized_name) DO NOTHING
	`, gameID, name, normalizedName, color); err != nil {
		return dbtypes.Tag{}, err
	}

	var tag dbtypes.Tag
	if err := tx.GetContext(ctx, &tag, `
		SELECT `+tagSelectColumns+`
		FROM tags
		WHERE game_id = ?
			AND normalized_name = ?
	`, gameID, normalizedName); err != nil {
		return dbtypes.Tag{}, err
	}
	return tag, nil
}

func getTagByID(ctx context.Context, db modGetter, tagID int64) (dbtypes.Tag, error) {
	var tag dbtypes.Tag
	if err := db.GetContext(ctx, &tag, `
		SELECT `+tagSelectColumns+`
		FROM tags
		WHERE id = ?
	`, tagID); err != nil {
		return dbtypes.Tag{}, err
	}
	return tag, nil
}

func validateTagName(name string) (string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", errors.New("tag name is required")
	}
	if utf8.RuneCountInString(name) > tagNameLimit {
		return "", "", fmt.Errorf("tag name must be %d characters or fewer", tagNameLimit)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", "", errors.New("tag name contains unsupported control characters")
		}
	}
	return name, strings.ToLower(name), nil
}

func validateTagColor(color dbtypes.TagColor) error {
	switch color {
	case dbtypes.TagColorRed,
		dbtypes.TagColorOrange,
		dbtypes.TagColorYellow,
		dbtypes.TagColorGreen,
		dbtypes.TagColorTeal,
		dbtypes.TagColorBlue,
		dbtypes.TagColorPurple,
		dbtypes.TagColorPink:
		return nil
	default:
		return fmt.Errorf("unsupported tag color %q", color)
	}
}
