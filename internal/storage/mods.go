package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

const modSelectColumns = `
	id,
	game_id,
	name,
	source_type,
	source_path,
	original_source_path,
	original_source_name,
	file_count,
	directory_count,
	total_size_bytes,
	metadata_json,
	created_at,
	updated_at
`

func (s *Store) ListMods(ctx context.Context, gameID int64) (mods []dbtypes.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select game mods: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}

	err = s.db.SelectContext(ctx, &mods, `
		SELECT `+modSelectColumns+`
		FROM mods
		WHERE game_id = ?
		ORDER BY LOWER(name), id
	`, gameID)
	if err != nil {
		return nil, err
	}

	return mods, nil
}

func (s *Store) GetMod(ctx context.Context, modID int64) (mod dbtypes.Mod, found bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select mod row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.Mod{}, false, errors.New("store is not open")
	}
	if modID <= 0 {
		return dbtypes.Mod{}, false, errors.New("mod ID must be positive")
	}

	mod, err = getModByID(ctx, s.db, modID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dbtypes.Mod{}, false, nil
		}

		return dbtypes.Mod{}, false, err
	}

	return mod, true, nil
}

func (s *Store) CreateMod(ctx context.Context, input dbtypes.CreateModInput) (mod dbtypes.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("insert mod row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.Mod{}, errors.New("store is not open")
	}

	mod, err = insertMod(ctx, s.db, input)
	if err != nil {
		return dbtypes.Mod{}, err
	}

	return mod, nil
}

func (s *Store) RenameMod(ctx context.Context, modID int64, name string) (mod dbtypes.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update mod name: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.Mod{}, errors.New("store is not open")
	}
	if modID <= 0 {
		return dbtypes.Mod{}, errors.New("mod ID must be positive")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return dbtypes.Mod{}, errors.New("mod name is required")
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE mods
		SET name = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, modID)
	if err != nil {
		return dbtypes.Mod{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return dbtypes.Mod{}, fmt.Errorf("get renamed mod count: %w", err)
	}
	if affected == 0 {
		return dbtypes.Mod{}, sql.ErrNoRows
	}

	return getModByID(ctx, s.db, modID)
}

func (s *Store) UpdateModPackage(ctx context.Context, input dbtypes.UpdateModPackageInput) (mod dbtypes.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update mod package row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.Mod{}, errors.New("store is not open")
	}
	if input.ModID <= 0 {
		return dbtypes.Mod{}, errors.New("mod ID must be positive")
	}

	err = withTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		updated, err := updateModPackage(ctx, tx, input)
		if err != nil {
			return err
		}
		if err := upsertDetectedModMetadata(ctx, tx, updated.ID, input.DetectedMetadata); err != nil {
			return fmt.Errorf("update detected mod metadata: %w", err)
		}

		mod = updated
		return nil
	})
	if err != nil {
		return dbtypes.Mod{}, err
	}

	return mod, nil
}

func (s *Store) DeleteMod(ctx context.Context, modID int64) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("delete mod row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}
	if modID <= 0 {
		return errors.New("mod ID must be positive")
	}

	result, err := s.db.ExecContext(ctx, `
		DELETE FROM mods
		WHERE id = ?
	`, modID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted mod count: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (s *Store) CountProfilesUsingMod(ctx context.Context, modID int64) (count int64, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("count profiles using mod: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return 0, errors.New("store is not open")
	}
	if modID <= 0 {
		return 0, errors.New("mod ID must be positive")
	}

	err = s.db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM profile_mods
		WHERE mod_id = ?
	`, modID)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Store) ProfileUsesMod(ctx context.Context, profileID int64, modID int64) (uses bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("check profile mod membership: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return false, errors.New("store is not open")
	}
	if profileID <= 0 {
		return false, errors.New("profile ID must be positive")
	}
	if modID <= 0 {
		return false, errors.New("mod ID must be positive")
	}

	var count int
	err = s.db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM profile_mods
		WHERE profile_id = ?
			AND mod_id = ?
	`, profileID, modID)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *Store) FindModByOriginalSourcePath(ctx context.Context, gameID int64, originalSourcePath string) (mod dbtypes.Mod, found bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("find mod by original source path: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.Mod{}, false, errors.New("store is not open")
	}

	originalSourcePath, err = CanonicalModOriginalSourcePath(originalSourcePath)
	if err != nil {
		return dbtypes.Mod{}, false, err
	}

	err = s.db.GetContext(ctx, &mod, `
		SELECT `+modSelectColumns+`
		FROM mods
		WHERE game_id = ?
			AND original_source_path = ?
	`, gameID, originalSourcePath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dbtypes.Mod{}, false, nil
		}

		return dbtypes.Mod{}, false, err
	}

	return mod, true, nil
}

func (s *Store) FindModsByOriginalSourcePaths(ctx context.Context, gameID int64, originalSourcePaths []string) (modsByPath map[string]dbtypes.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("find mods by original source paths: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}
	if gameID <= 0 {
		return nil, errors.New("game ID must be positive")
	}
	if len(originalSourcePaths) == 0 {
		return map[string]dbtypes.Mod{}, nil
	}

	canonicalPaths := make([]string, 0, len(originalSourcePaths))
	seen := make(map[string]struct{}, len(originalSourcePaths))
	for _, originalSourcePath := range originalSourcePaths {
		canonicalPath, canonicalErr := CanonicalModOriginalSourcePath(originalSourcePath)
		if canonicalErr != nil {
			return nil, canonicalErr
		}
		if _, found := seen[canonicalPath]; found {
			continue
		}
		seen[canonicalPath] = struct{}{}
		canonicalPaths = append(canonicalPaths, canonicalPath)
	}

	query, args, err := sqlx.In(`
		SELECT `+modSelectColumns+`
		FROM mods
		WHERE game_id = ?
			AND original_source_path IN (?)
	`, gameID, canonicalPaths)
	if err != nil {
		return nil, err
	}
	query = s.db.Rebind(query)

	mods := make([]dbtypes.Mod, 0, len(canonicalPaths))
	if err := s.db.SelectContext(ctx, &mods, query, args...); err != nil {
		return nil, err
	}

	modsByPath = make(map[string]dbtypes.Mod, len(mods))
	for _, mod := range mods {
		modsByPath[mod.OriginalSourcePath] = mod
	}

	return modsByPath, nil
}

func insertMod(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	GetContext(context.Context, any, string, ...any) error
}, input dbtypes.CreateModInput) (dbtypes.Mod, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return dbtypes.Mod{}, errors.New("mod name is required")
	}

	sourceType := input.SourceType
	if sourceType == "" {
		sourceType = dbtypes.ModSourceTypeFolder
	}
	if sourceType != dbtypes.ModSourceTypeFolder && sourceType != dbtypes.ModSourceTypeArchive {
		return dbtypes.Mod{}, fmt.Errorf("unsupported mod source type %q", sourceType)
	}

	sourcePath := cleanOptionalPath(input.SourcePath)
	if sourcePath == "" {
		return dbtypes.Mod{}, errors.New("managed mod source path is required")
	}

	originalSourcePath, err := CanonicalModOriginalSourcePath(input.OriginalSourcePath)
	if err != nil {
		return dbtypes.Mod{}, err
	}

	originalSourceName := cleanOptionalString(input.OriginalSourceName)
	metadataJSON := cleanOptionalString(input.MetadataJSON)
	result, err := db.ExecContext(ctx, `
		INSERT INTO mods (
			game_id,
			name,
			source_type,
			source_path,
			original_source_path,
			original_source_name,
			file_count,
			directory_count,
			total_size_bytes,
			metadata_json
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, input.GameID, name, sourceType, sourcePath, originalSourcePath, nullableText(originalSourceName), input.FileCount, input.DirectoryCount, input.TotalSizeBytes, nullableText(metadataJSON))
	if err != nil {
		return dbtypes.Mod{}, err
	}

	modID, err := result.LastInsertId()
	if err != nil {
		return dbtypes.Mod{}, fmt.Errorf("get created mod id: %w", err)
	}
	if err := upsertDetectedModMetadata(ctx, db, modID, input.DetectedMetadata); err != nil {
		return dbtypes.Mod{}, fmt.Errorf("insert mod metadata row: %w", err)
	}

	return getModByID(ctx, db, modID)
}

func updateModPackage(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	GetContext(context.Context, any, string, ...any) error
}, input dbtypes.UpdateModPackageInput) (dbtypes.Mod, error) {
	if input.ModID <= 0 {
		return dbtypes.Mod{}, errors.New("mod ID must be positive")
	}

	sourceType := input.SourceType
	if sourceType == "" {
		sourceType = dbtypes.ModSourceTypeFolder
	}
	if sourceType != dbtypes.ModSourceTypeFolder && sourceType != dbtypes.ModSourceTypeArchive {
		return dbtypes.Mod{}, fmt.Errorf("unsupported mod source type %q", sourceType)
	}

	originalSourcePath, err := CanonicalModOriginalSourcePath(input.OriginalSourcePath)
	if err != nil {
		return dbtypes.Mod{}, err
	}

	originalSourceName := cleanOptionalString(input.OriginalSourceName)
	metadataJSON := cleanOptionalString(input.MetadataJSON)
	result, err := db.ExecContext(ctx, `
		UPDATE mods
		SET source_type = ?,
			original_source_path = ?,
			original_source_name = ?,
			file_count = ?,
			directory_count = ?,
			total_size_bytes = ?,
			metadata_json = ?,
			updated_at = strftime('%Y-%m-%d %H:%M:%f', 'now')
		WHERE id = ?
	`, sourceType, originalSourcePath, nullableText(originalSourceName), input.FileCount, input.DirectoryCount, input.TotalSizeBytes, nullableText(metadataJSON), input.ModID)
	if err != nil {
		return dbtypes.Mod{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return dbtypes.Mod{}, fmt.Errorf("get updated mod package count: %w", err)
	}
	if affected == 0 {
		return dbtypes.Mod{}, sql.ErrNoRows
	}

	return getModByID(ctx, db, input.ModID)
}

type modGetter interface {
	GetContext(context.Context, any, string, ...any) error
}

func getModByID(ctx context.Context, db modGetter, modID int64) (dbtypes.Mod, error) {
	var mod dbtypes.Mod
	err := db.GetContext(ctx, &mod, `
		SELECT `+modSelectColumns+`
		FROM mods
		WHERE id = ?
	`, modID)
	if err != nil {
		return dbtypes.Mod{}, err
	}

	return mod, nil
}

func CanonicalModOriginalSourcePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("original source path is required")
	}

	path = filepath.Clean(path)
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return path, nil
	}

	return filepath.Clean(absolutePath), nil
}
