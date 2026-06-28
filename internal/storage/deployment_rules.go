package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

const deploymentRuleKindPerFileWinner = "per_file_winner"

func (s *Store) ListDeploymentRulesByProfileID(
	ctx context.Context,
	profileID int64,
) (rules []dbtypes.DeploymentRuleRow, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list deployment rules by profile ID: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return nil, errors.New("store is not open")
	}
	if profileID <= 0 {
		return nil, errors.New("profile ID must be positive")
	}

	err = s.db.SelectContext(ctx, &rules, `
		SELECT
			id,
			profile_id,
			game_relative_path,
			rule_kind,
			winner_mod_id,
			explanation,
			created_at
		FROM deployment_rules
		WHERE profile_id = ?
		ORDER BY game_relative_path COLLATE NOCASE
	`, profileID)
	if err != nil {
		return nil, err
	}
	if rules == nil {
		rules = []dbtypes.DeploymentRuleRow{}
	}

	return rules, nil
}

func (s *Store) UpsertPerFileWinnerRule(
	ctx context.Context,
	profileID int64,
	gameRelativePath string,
	winnerModID int64,
) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("upsert per-file winner rule: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}
	if profileID <= 0 {
		return errors.New("profile ID must be positive")
	}
	if strings.TrimSpace(gameRelativePath) == "" {
		return errors.New("game relative path is required")
	}
	if winnerModID <= 0 {
		return errors.New("winner mod ID must be positive")
	}

	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO deployment_rules (
			profile_id,
			game_relative_path,
			rule_kind,
			winner_mod_id,
			created_at
		)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(profile_id, game_relative_path, rule_kind) DO UPDATE SET
			winner_mod_id = excluded.winner_mod_id,
			created_at = excluded.created_at
	`, profileID, gameRelativePath, deploymentRuleKindPerFileWinner, winnerModID, createdAt)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) DeletePerFileWinnerRule(
	ctx context.Context,
	profileID int64,
	gameRelativePath string,
) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("delete per-file winner rule: %w", err)
		}
	}()

	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}
	if profileID <= 0 {
		return errors.New("profile ID must be positive")
	}
	if strings.TrimSpace(gameRelativePath) == "" {
		return errors.New("game relative path is required")
	}

	_, err = s.db.ExecContext(ctx, `
		DELETE FROM deployment_rules
		WHERE profile_id = ?
			AND game_relative_path = ?
			AND rule_kind = ?
	`, profileID, gameRelativePath, deploymentRuleKindPerFileWinner)
	if err != nil {
		return err
	}

	return nil
}
