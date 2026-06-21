package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

const optiScalerTargetColumns = `
	t.id, t.game_id, t.target_relative_path, t.executable_relative_path, t.api_family AS graphics_api,
	o.proxy_filename, o.dxgi_spoofing, o.process_filter, o.release_tag, o.release_version,
	o.release_asset_name, o.release_digest, o.management_origin, t.status, o.manifest_json,
	o.warning_version, o.warning_acknowledged_at, t.created_at, t.updated_at, t.last_verified_at
`

func (s *Store) SaveOptiScalerTarget(ctx context.Context, input dbtypes.SaveOptiScalerTargetInput) (target dbtypes.OptiScalerTarget, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("upsert OptiScaler target row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.OptiScalerTarget{}, errors.New("store is not open")
	}
	input, err = validateSaveOptiScalerTargetInput(input)
	if err != nil {
		return dbtypes.OptiScalerTarget{}, err
	}

	targetID, err := s.saveInjectionTarget(ctx, injectionTargetInput{
		GameID:                 input.GameID,
		TargetRelativePath:     input.TargetRelativePath,
		ExecutableRelativePath: input.ExecutableRelativePath,
		APIFamily:              input.GraphicsAPI,
		Architecture:           "x64",
		PrimaryOwner:           "optiscaler",
		PrimaryProxyFilename:   input.ProxyFilename,
		Status:                 input.Status,
		LastVerifiedAt:         input.LastVerifiedAt,
	})
	if err != nil {
		return dbtypes.OptiScalerTarget{}, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO injection_optiscaler (
			injection_target_id, proxy_filename, dxgi_spoofing, process_filter,
			release_tag, release_version, release_asset_name, release_digest,
			management_origin, manifest_json, warning_version, warning_acknowledged_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(injection_target_id) DO UPDATE SET
			proxy_filename = excluded.proxy_filename,
			dxgi_spoofing = excluded.dxgi_spoofing,
			process_filter = excluded.process_filter,
			release_tag = excluded.release_tag,
			release_version = excluded.release_version,
			release_asset_name = excluded.release_asset_name,
			release_digest = excluded.release_digest,
			management_origin = excluded.management_origin,
			manifest_json = excluded.manifest_json,
			warning_version = excluded.warning_version,
			warning_acknowledged_at = excluded.warning_acknowledged_at
	`, targetID, input.ProxyFilename, input.DXGISpoofing, nullableText(cleanOptionalString(input.ProcessFilter)),
		input.ReleaseTag, input.ReleaseVersion, input.ReleaseAssetName, input.ReleaseDigest,
		input.ManagementOrigin, input.ManifestJSON, input.WarningVersion,
		nullableText(cleanOptionalString(input.WarningAcknowledgedAt)))
	if err != nil {
		return dbtypes.OptiScalerTarget{}, err
	}

	target, found, err := s.GetOptiScalerTarget(ctx, input.GameID, input.TargetRelativePath)
	if err != nil {
		return dbtypes.OptiScalerTarget{}, err
	}
	if !found {
		return dbtypes.OptiScalerTarget{}, sql.ErrNoRows
	}
	return target, nil
}

func (s *Store) GetOptiScalerTarget(ctx context.Context, gameID int64, targetRelativePath string) (target dbtypes.OptiScalerTarget, found bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select OptiScaler target row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return dbtypes.OptiScalerTarget{}, false, errors.New("store is not open")
	}
	targetRelativePath, err = cleanRelativePath("target relative path", targetRelativePath)
	if err != nil {
		return dbtypes.OptiScalerTarget{}, false, err
	}

	err = s.db.GetContext(ctx, &target, `
		SELECT `+optiScalerTargetColumns+`
		FROM injection_targets t
		JOIN injection_optiscaler o ON o.injection_target_id = t.id
		WHERE t.game_id = ? AND t.target_relative_path = ? COLLATE NOCASE
	`, gameID, targetRelativePath)
	if errors.Is(err, sql.ErrNoRows) {
		return dbtypes.OptiScalerTarget{}, false, nil
	}
	if err != nil {
		return dbtypes.OptiScalerTarget{}, false, err
	}
	return target, true, nil
}

func (s *Store) ListOptiScalerTargets(ctx context.Context, gameID int64) (targets []dbtypes.OptiScalerTarget, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list OptiScaler target rows: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}
	err = s.db.SelectContext(ctx, &targets, `
		SELECT `+optiScalerTargetColumns+`
		FROM injection_targets t
		JOIN injection_optiscaler o ON o.injection_target_id = t.id
		WHERE t.game_id = ?
		ORDER BY t.target_relative_path COLLATE NOCASE, t.id
	`, gameID)
	return targets, err
}

func (s *Store) DeleteOptiScalerTarget(ctx context.Context, gameID int64, targetRelativePath string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("delete OptiScaler target row: %w", err)
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
		DELETE FROM injection_optiscaler
		WHERE injection_target_id = ?
	`, targetID)
	if err != nil {
		return err
	}
	return s.deleteInjectionTargetIfEmpty(ctx, gameID, targetRelativePath)
}

func validateSaveOptiScalerTargetInput(input dbtypes.SaveOptiScalerTargetInput) (dbtypes.SaveOptiScalerTargetInput, error) {
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
	if input.GraphicsAPI != "directx" && input.GraphicsAPI != "vulkan" {
		return input, errors.New("graphics API must be directx or vulkan")
	}
	if strings.TrimSpace(input.ProxyFilename) == "" {
		return input, errors.New("proxy filename is required")
	}
	if input.ManagementOrigin != "installed" && input.ManagementOrigin != "adopted" {
		return input, errors.New("management origin must be installed or adopted")
	}
	if input.Status != "managed" && input.Status != "drifted" && input.Status != "recovery_required" {
		return input, errors.New("status must be managed, drifted, or recovery_required")
	}
	for name, value := range map[string]string{
		"release tag": input.ReleaseTag, "release version": input.ReleaseVersion,
		"release asset name": input.ReleaseAssetName, "release digest": input.ReleaseDigest,
		"warning version": input.WarningVersion,
	} {
		if strings.TrimSpace(value) == "" {
			return input, fmt.Errorf("%s is required", name)
		}
	}
	input.ManifestJSON = strings.TrimSpace(input.ManifestJSON)
	if input.ManifestJSON == "" || !json.Valid([]byte(input.ManifestJSON)) {
		return input, errors.New("manifest JSON is required and must be valid")
	}
	return input, nil
}

func cleanRelativePath(name string, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	value = filepath.Clean(value)
	if filepath.IsAbs(value) || value == ".." || strings.HasPrefix(value, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s must stay relative", name)
	}
	if value == "." {
		return ".", nil
	}
	return value, nil
}
