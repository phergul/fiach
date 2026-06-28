package storage

import (
	"context"
	"testing"
)

func TestMigrateUpAddsDeploymentRulesTable(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if !tableExists(t, store, "deployment_rules") {
		t.Fatal("expected deployment_rules table to exist")
	}
	for _, column := range []string{
		"id",
		"profile_id",
		"game_relative_path",
		"rule_kind",
		"winner_mod_id",
		"explanation",
		"created_at",
	} {
		if !columnExists(t, store, "deployment_rules", column) {
			t.Fatalf("expected deployment_rules.%s column to exist", column)
		}
	}
	if !indexExists(t, store, "idx_deployment_rules_profile_id") {
		t.Fatal("expected idx_deployment_rules_profile_id to exist")
	}
}

func TestUpsertAndDeletePerFileWinnerRule(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")

	if err := store.UpsertPerFileWinnerRule(context.Background(), profile.ID, "Data/SkyUI.esp", 10); err != nil {
		t.Fatalf("UpsertPerFileWinnerRule() error = %v", err)
	}

	rules, err := store.ListDeploymentRulesByProfileID(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("ListDeploymentRulesByProfileID() error = %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("rule count = %d, want 1", len(rules))
	}
	if rules[0].WinnerModID == nil || *rules[0].WinnerModID != 10 {
		t.Fatalf("winner mod ID = %+v, want 10", rules[0].WinnerModID)
	}

	if err := store.UpsertPerFileWinnerRule(context.Background(), profile.ID, "data/skyui.esp", 20); err != nil {
		t.Fatalf("UpsertPerFileWinnerRule(replace) error = %v", err)
	}

	rules, err = store.ListDeploymentRulesByProfileID(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("ListDeploymentRulesByProfileID(replace) error = %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("rule count after replace = %d, want 1", len(rules))
	}
	if rules[0].WinnerModID == nil || *rules[0].WinnerModID != 20 {
		t.Fatalf("winner mod ID after replace = %+v, want 20", rules[0].WinnerModID)
	}

	if err := store.DeletePerFileWinnerRule(context.Background(), profile.ID, "Data/SkyUI.esp"); err != nil {
		t.Fatalf("DeletePerFileWinnerRule() error = %v", err)
	}

	rules, err = store.ListDeploymentRulesByProfileID(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("ListDeploymentRulesByProfileID(after delete) error = %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("rule count after delete = %d, want 0", len(rules))
	}
}
