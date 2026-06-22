package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

const reShadeTargetColumns = `
	t.id, t.game_id, t.target_relative_path, t.executable_relative_path,
	COALESCE(t.directx_api, CASE LOWER(r.preferred_proxy_filename)
		WHEN 'd3d9.dll' THEN 'd3d9'
		WHEN 'd3d10.dll' THEN 'd3d10'
		WHEN 'd3d10core.dll' THEN 'd3d10'
		WHEN 'd3d11.dll' THEN 'd3d11'
		WHEN 'd3d12.dll' THEN 'd3d12'
		WHEN 'dxgi.dll' THEN 'd3d11'
	END) AS rendering_api,
	r.preferred_proxy_filename AS proxy_filename, t.architecture, r.build_variant, r.runtime_version, r.installer_tag,
	r.installer_asset_name, r.installer_url, r.installer_digest, r.installer_size,
	r.management_origin, t.status, r.manifest_json, t.created_at, t.updated_at, t.last_verified_at
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
	primaryOwner := "reshade"
	primaryProxyFilename := input.ProxyFilename
	if existingProxy, found, proxyErr := s.optiscalerProxyForInjectionTarget(ctx, input.GameID, input.TargetRelativePath); proxyErr != nil {
		return dbtypes.ReShadeTarget{}, proxyErr
	} else if found {
		primaryOwner = "optiscaler"
		primaryProxyFilename = existingProxy
	}
	directXAPI := input.RenderingAPI
	targetID, err := s.saveInjectionTarget(ctx, injectionTargetInput{
		GameID:                 input.GameID,
		TargetRelativePath:     input.TargetRelativePath,
		ExecutableRelativePath: input.ExecutableRelativePath,
		APIFamily:              "directx",
		DirectXAPI:             &directXAPI,
		Architecture:           input.Architecture,
		PrimaryOwner:           primaryOwner,
		PrimaryProxyFilename:   primaryProxyFilename,
		Status:                 input.Status,
		LastVerifiedAt:         input.LastVerifiedAt,
	})
	if err != nil {
		return dbtypes.ReShadeTarget{}, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO injection_reshade (
			injection_target_id, preferred_proxy_filename, active_runtime_filename,
			build_variant, runtime_version, installer_tag, installer_asset_name,
			installer_url, installer_digest, installer_size, management_origin,
			manifest_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(injection_target_id) DO UPDATE SET
			preferred_proxy_filename = excluded.preferred_proxy_filename,
			active_runtime_filename = excluded.active_runtime_filename,
			build_variant = excluded.build_variant,
			runtime_version = excluded.runtime_version,
			installer_tag = excluded.installer_tag,
			installer_asset_name = excluded.installer_asset_name,
			installer_url = excluded.installer_url,
			installer_digest = excluded.installer_digest,
			installer_size = excluded.installer_size,
			management_origin = excluded.management_origin,
			manifest_json = excluded.manifest_json
	`, targetID, input.ProxyFilename, input.ProxyFilename, input.BuildVariant, input.RuntimeVersion,
		nullableText(cleanOptionalString(input.InstallerTag)), nullableText(cleanOptionalString(input.InstallerAssetName)),
		nullableText(cleanOptionalString(input.InstallerURL)), nullableText(cleanOptionalString(input.InstallerDigest)),
		input.InstallerSize, input.ManagementOrigin, input.ManifestJSON)
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
		FROM injection_targets t
		JOIN injection_reshade r ON r.injection_target_id = t.id
		WHERE t.game_id = ? AND t.target_relative_path = ? COLLATE NOCASE
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
		FROM injection_targets t
		JOIN injection_reshade r ON r.injection_target_id = t.id
		WHERE t.game_id = ?
		ORDER BY t.target_relative_path COLLATE NOCASE, t.id
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
	targetID, err := s.injectionTargetID(ctx, gameID, targetRelativePath)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		DELETE FROM injection_reshade
		WHERE injection_target_id = ?
	`, targetID)
	if err != nil {
		return err
	}
	return s.deleteInjectionTargetIfEmpty(ctx, gameID, targetRelativePath)
}

func (s *Store) optiscalerProxyForInjectionTarget(ctx context.Context, gameID int64, targetRelativePath string) (string, bool, error) {
	targetID, err := s.injectionTargetID(ctx, gameID, targetRelativePath)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	var proxy string
	err = s.db.GetContext(ctx, &proxy, `
		SELECT proxy_filename
		FROM injection_optiscaler
		WHERE injection_target_id = ?
	`, targetID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("select OptiScaler injection target proxy: %w", err)
	}
	return proxy, true, nil
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
	if !slices.Contains([]string{"d3d9", "d3d10", "d3d11", "d3d12"}, input.RenderingAPI) {
		return input, errors.New("rendering API is invalid")
	}
	if strings.TrimSpace(input.ProxyFilename) == "" {
		return input, errors.New("proxy filename is required")
	}
	if !slices.Contains([]string{"x86", "x64"}, input.Architecture) {
		return input, errors.New("architecture is invalid")
	}
	if !slices.Contains([]string{"standard", "addon"}, input.BuildVariant) {
		return input, errors.New("build variant is invalid")
	}
	if strings.TrimSpace(input.RuntimeVersion) == "" {
		return input, errors.New("runtime version is required")
	}
	if input.InstallerSize != nil && *input.InstallerSize < 0 {
		return input, errors.New("installer size cannot be negative")
	}
	if !slices.Contains([]string{"installed", "adopted"}, input.ManagementOrigin) {
		return input, errors.New("management origin is invalid")
	}
	if !slices.Contains([]string{"managed", "drifted", "recovery_required"}, input.Status) {
		return input, errors.New("status is invalid")
	}
	input.ManifestJSON = strings.TrimSpace(input.ManifestJSON)
	if input.ManifestJSON == "" || !json.Valid([]byte(input.ManifestJSON)) {
		return input, errors.New("manifest JSON is required and must be valid")
	}
	return input, nil
}
