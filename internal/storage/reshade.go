package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

const reShadeTargetColumns = `
	id, game_id, target_relative_path, executable_relative_path, rendering_api,
	proxy_filename, architecture, build_variant, runtime_version, installer_tag,
	installer_asset_name, installer_url, installer_digest, installer_size,
	management_origin, status, manifest_json, created_at, updated_at, last_verified_at
`

func (s *Store) SaveReShadeTarget(ctx context.Context, input dbtypes.SaveReShadeTargetInput) (target dbtypes.ReShadeTarget, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("upsert ReShade target row: %w", err)
		}
	}()
	if s == nil || s.db == nil {
		return dbtypes.ReShadeTarget{}, errors.New("store is not open")
	}
	input, err = validateSaveReShadeTargetInput(input)
	if err != nil {
		return dbtypes.ReShadeTarget{}, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO reshade_targets (
			game_id, target_relative_path, executable_relative_path, rendering_api,
			proxy_filename, architecture, build_variant, runtime_version, installer_tag,
			installer_asset_name, installer_url, installer_digest, installer_size,
			management_origin, status, manifest_json, last_verified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(game_id, target_relative_path) DO UPDATE SET
			executable_relative_path = excluded.executable_relative_path,
			rendering_api = excluded.rendering_api,
			proxy_filename = excluded.proxy_filename,
			architecture = excluded.architecture,
			build_variant = excluded.build_variant,
			runtime_version = excluded.runtime_version,
			installer_tag = excluded.installer_tag,
			installer_asset_name = excluded.installer_asset_name,
			installer_url = excluded.installer_url,
			installer_digest = excluded.installer_digest,
			installer_size = excluded.installer_size,
			management_origin = excluded.management_origin,
			status = excluded.status,
			manifest_json = excluded.manifest_json,
			last_verified_at = excluded.last_verified_at,
			updated_at = CURRENT_TIMESTAMP
	`, input.GameID, input.TargetRelativePath, input.ExecutableRelativePath, input.RenderingAPI,
		input.ProxyFilename, input.Architecture, input.BuildVariant, input.RuntimeVersion,
		nullableText(cleanOptionalString(input.InstallerTag)), nullableText(cleanOptionalString(input.InstallerAssetName)),
		nullableText(cleanOptionalString(input.InstallerURL)), nullableText(cleanOptionalString(input.InstallerDigest)),
		input.InstallerSize, input.ManagementOrigin, input.Status, input.ManifestJSON,
		nullableText(cleanOptionalString(input.LastVerifiedAt)))
	if err != nil {
		return dbtypes.ReShadeTarget{}, err
	}
	target, found, err := s.GetReShadeTarget(ctx, input.GameID, input.TargetRelativePath)
	if err != nil {
		return dbtypes.ReShadeTarget{}, err
	}
	if !found {
		return dbtypes.ReShadeTarget{}, sql.ErrNoRows
	}
	return target, nil
}

func (s *Store) GetReShadeTarget(ctx context.Context, gameID int64, targetRelativePath string) (target dbtypes.ReShadeTarget, found bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select ReShade target row: %w", err)
		}
	}()
	if s == nil || s.db == nil {
		return dbtypes.ReShadeTarget{}, false, errors.New("store is not open")
	}
	targetRelativePath, err = cleanRelativePath("target relative path", targetRelativePath)
	if err != nil {
		return dbtypes.ReShadeTarget{}, false, err
	}
	err = s.db.GetContext(ctx, &target, `
		SELECT `+reShadeTargetColumns+`
		FROM reshade_targets
		WHERE game_id = ? AND target_relative_path = ? COLLATE NOCASE
	`, gameID, targetRelativePath)
	if errors.Is(err, sql.ErrNoRows) {
		return dbtypes.ReShadeTarget{}, false, nil
	}
	if err != nil {
		return dbtypes.ReShadeTarget{}, false, err
	}
	return target, true, nil
}

func (s *Store) ListReShadeTargets(ctx context.Context, gameID int64) (targets []dbtypes.ReShadeTarget, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list ReShade target rows: %w", err)
		}
	}()
	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}
	err = s.db.SelectContext(ctx, &targets, `
		SELECT `+reShadeTargetColumns+`
		FROM reshade_targets
		WHERE game_id = ?
		ORDER BY target_relative_path COLLATE NOCASE, id
	`, gameID)
	return targets, err
}

func (s *Store) DeleteReShadeTarget(ctx context.Context, gameID int64, targetRelativePath string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("delete ReShade target row: %w", err)
		}
	}()
	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}
	targetRelativePath, err = cleanRelativePath("target relative path", targetRelativePath)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		DELETE FROM reshade_targets
		WHERE game_id = ? AND target_relative_path = ? COLLATE NOCASE
	`, gameID, targetRelativePath)
	return err
}

func validateSaveReShadeTargetInput(input dbtypes.SaveReShadeTargetInput) (dbtypes.SaveReShadeTargetInput, error) {
	var err error
	if input.GameID <= 0 {
		return input, errors.New("game ID must be positive")
	}
	input.TargetRelativePath, err = cleanRelativePath("target relative path", input.TargetRelativePath)
	if err != nil {
		return input, err
	}
	input.ExecutableRelativePath, err = cleanRelativePath("executable relative path", input.ExecutableRelativePath)
	if err != nil {
		return input, err
	}
	if !oneOf(input.RenderingAPI, "d3d9", "d3d10", "d3d11", "d3d12") {
		return input, errors.New("rendering API is invalid")
	}
	if strings.TrimSpace(input.ProxyFilename) == "" {
		return input, errors.New("proxy filename is required")
	}
	if !oneOf(input.Architecture, "x86", "x64") {
		return input, errors.New("architecture is invalid")
	}
	if !oneOf(input.BuildVariant, "standard", "addon") {
		return input, errors.New("build variant is invalid")
	}
	if strings.TrimSpace(input.RuntimeVersion) == "" {
		return input, errors.New("runtime version is required")
	}
	if input.InstallerSize != nil && *input.InstallerSize < 0 {
		return input, errors.New("installer size cannot be negative")
	}
	if !oneOf(input.ManagementOrigin, "installed", "adopted") {
		return input, errors.New("management origin is invalid")
	}
	if !oneOf(input.Status, "managed", "drifted", "recovery_required") {
		return input, errors.New("status is invalid")
	}
	input.ManifestJSON = strings.TrimSpace(input.ManifestJSON)
	if input.ManifestJSON == "" || !json.Valid([]byte(input.ManifestJSON)) {
		return input, errors.New("manifest JSON is required and must be valid")
	}
	return input, nil
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
