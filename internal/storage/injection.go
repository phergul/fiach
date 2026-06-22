package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
)

type injectionTargetInput struct {
	GameID                 int64
	TargetRelativePath     string
	ExecutableRelativePath string
	APIFamily              string
	DirectXAPI             *string
	Architecture           string
	PrimaryOwner           string
	PrimaryProxyFilename   string
	Status                 string
	LastVerifiedAt         *string
}

func (s *Store) saveInjectionTarget(ctx context.Context, input injectionTargetInput) (id int64, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("upsert injection target row: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return 0, errors.New("store is not open")
	}
	input, err = validateInjectionTargetInput(input)
	if err != nil {
		return 0, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO injection_targets (
			game_id, target_relative_path, executable_relative_path, api_family,
			directx_api, architecture, primary_owner, primary_proxy_filename,
			status, last_verified_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(game_id, target_relative_path) DO UPDATE SET
			executable_relative_path = excluded.executable_relative_path,
			api_family = excluded.api_family,
			directx_api = CASE
				WHEN excluded.api_family = 'directx' THEN COALESCE(excluded.directx_api, directx_api)
				ELSE NULL
			END,
			architecture = excluded.architecture,
			primary_owner = excluded.primary_owner,
			primary_proxy_filename = excluded.primary_proxy_filename,
			status = excluded.status,
			last_verified_at = excluded.last_verified_at,
			updated_at = CURRENT_TIMESTAMP
	`, input.GameID, input.TargetRelativePath, input.ExecutableRelativePath, input.APIFamily,
		input.DirectXAPI, input.Architecture, input.PrimaryOwner, input.PrimaryProxyFilename,
		input.Status, nullableText(cleanOptionalString(input.LastVerifiedAt)))
	if err != nil {
		return 0, err
	}
	id, err = s.injectionTargetID(ctx, input.GameID, input.TargetRelativePath)
	return id, err
}

func (s *Store) injectionTargetID(ctx context.Context, gameID int64, targetRelativePath string) (id int64, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("select injection target ID: %w", err)
		}
	}()

	targetRelativePath, err = cleanRelativePath("target relative path", targetRelativePath)
	if err != nil {
		return 0, err
	}
	err = s.db.GetContext(ctx, &id, `
		SELECT id
		FROM injection_targets
		WHERE game_id = ? AND target_relative_path = ? COLLATE NOCASE
	`, gameID, targetRelativePath)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, sql.ErrNoRows
	}
	return id, err
}

func (s *Store) deleteInjectionTargetIfEmpty(ctx context.Context, gameID int64, targetRelativePath string) error {
	id, err := s.injectionTargetID(ctx, gameID, targetRelativePath)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var detailCount int
	if err := s.db.GetContext(ctx, &detailCount, `
		SELECT
			(SELECT COUNT(*) FROM injection_optiscaler WHERE injection_target_id = ?) +
			(SELECT COUNT(*) FROM injection_reshade WHERE injection_target_id = ?)
	`, id, id); err != nil {
		return fmt.Errorf("count injection target product rows: %w", err)
	}
	if detailCount > 0 {
		return nil
	}
	_, err = s.db.ExecContext(ctx, `
		DELETE FROM injection_targets
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("delete empty injection target row: %w", err)
	}
	return nil
}

func validateInjectionTargetInput(input injectionTargetInput) (injectionTargetInput, error) {
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
	if !slices.Contains([]string{"directx", "vulkan"}, input.APIFamily) {
		return input, errors.New("API family is invalid")
	}
	if input.DirectXAPI != nil && !slices.Contains([]string{"d3d9", "d3d10", "d3d11", "d3d12"}, *input.DirectXAPI) {
		return input, errors.New("DirectX API is invalid")
	}
	if input.APIFamily != "directx" {
		input.DirectXAPI = nil
	}
	if !slices.Contains([]string{"x86", "x64"}, input.Architecture) {
		return input, errors.New("architecture is invalid")
	}
	if !slices.Contains([]string{"reshade", "optiscaler"}, input.PrimaryOwner) {
		return input, errors.New("primary owner is invalid")
	}
	if strings.TrimSpace(input.PrimaryProxyFilename) == "" {
		return input, errors.New("primary proxy filename is required")
	}
	if !slices.Contains([]string{"managed", "drifted", "recovery_required"}, input.Status) {
		return input, errors.New("status is invalid")
	}
	return input, nil
}
