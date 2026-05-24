package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/phergul/mod-manager/internal/storage/dbtypes"
)

func (s *Store) CreateOrReplaceModInstallConfig(ctx context.Context, input dbtypes.CreateModInstallConfigInput) (config dbtypes.ModInstallConfig, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("upsert mod install config row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.ModInstallConfig{}, errors.New("store is not open")
	}

	if err := validateModInstallConfigInput(input); err != nil {
		return dbtypes.ModInstallConfig{}, err
	}

	config, err = upsertModInstallConfig(ctx, s.db, input)
	if err != nil {
		return dbtypes.ModInstallConfig{}, err
	}

	return config, nil
}

func (s *Store) GetModInstallConfig(ctx context.Context, modID int64) (config dbtypes.ModInstallConfig, found bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select mod install config: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.ModInstallConfig{}, false, errors.New("store is not open")
	}

	config, found, err = getModInstallConfig(ctx, s.db, modID)
	if err != nil {
		return dbtypes.ModInstallConfig{}, false, err
	}

	return config, found, nil
}

func (s *Store) CreateModWithInstallConfig(ctx context.Context, input dbtypes.CreateModWithInstallConfigInput) (result dbtypes.CreateModWithInstallConfigResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("insert mod with install config rows: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.CreateModWithInstallConfigResult{}, errors.New("store is not open")
	}

	err = withTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		mod, err := insertMod(ctx, tx, input.Mod)
		if err != nil {
			return err
		}

		configInput := input.Config
		configInput.ModID = mod.ID
		config, err := upsertModInstallConfig(ctx, tx, configInput)
		if err != nil {
			return err
		}

		result = dbtypes.CreateModWithInstallConfigResult{
			Mod:    mod,
			Config: config,
		}
		return nil
	})
	if err != nil {
		return dbtypes.CreateModWithInstallConfigResult{}, err
	}

	return result, nil
}

func upsertModInstallConfig(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	GetContext(context.Context, any, string, ...any) error
}, input dbtypes.CreateModInstallConfigInput) (dbtypes.ModInstallConfig, error) {
	if err := validateModInstallConfigInput(input); err != nil {
		return dbtypes.ModInstallConfig{}, err
	}

	sourceSubpath := cleanOptionalString(input.SourceSubpath)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO mod_install_configs (
			mod_id,
			strategy_type,
			target_base,
			target_relative_path,
			source_subpath
		)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(mod_id) DO UPDATE SET
			strategy_type = excluded.strategy_type,
			target_base = excluded.target_base,
			target_relative_path = excluded.target_relative_path,
			source_subpath = excluded.source_subpath,
			updated_at = CURRENT_TIMESTAMP
	`, input.ModID, strings.TrimSpace(input.StrategyType), strings.TrimSpace(input.TargetBase), strings.TrimSpace(input.TargetRelativePath), nullableText(sourceSubpath)); err != nil {
		return dbtypes.ModInstallConfig{}, err
	}

	config, found, err := getModInstallConfig(ctx, db, input.ModID)
	if err != nil {
		return dbtypes.ModInstallConfig{}, err
	}
	if !found {
		return dbtypes.ModInstallConfig{}, sql.ErrNoRows
	}

	return config, nil
}

func validateModInstallConfigInput(input dbtypes.CreateModInstallConfigInput) error {
	if input.ModID <= 0 {
		return errors.New("mod id is required")
	}
	if strings.TrimSpace(input.StrategyType) == "" {
		return errors.New("install strategy type is required")
	}
	if strings.TrimSpace(input.TargetBase) == "" {
		return errors.New("install target base is required")
	}
	if strings.TrimSpace(input.TargetRelativePath) == "" {
		return errors.New("install target relative path is required")
	}

	return nil
}

func getModInstallConfig(ctx context.Context, db modGetter, modID int64) (dbtypes.ModInstallConfig, bool, error) {
	var config dbtypes.ModInstallConfig
	err := db.GetContext(ctx, &config, `
		SELECT mod_id, strategy_type, target_base, target_relative_path, source_subpath, created_at, updated_at
		FROM mod_install_configs
		WHERE mod_id = ?
	`, modID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dbtypes.ModInstallConfig{}, false, nil
		}

		return dbtypes.ModInstallConfig{}, false, err
	}

	return config, true, nil
}
