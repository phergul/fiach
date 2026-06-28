package rules_test

import (
	"testing"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/rules"
)

func TestApplyPerFileWinnerOverridesLoadOrderContent(t *testing.T) {
	t.Parallel()

	alphaID := int64(1)
	betaID := int64(2)
	file := deployment.DesiredFile{
		SHA256:     "beta-hash",
		SourcePath: "/mods/beta/plugin.txt",
		SizeBytes:  4,
		ModContentByID: map[int64]deployment.ModFileContent{
			alphaID: {
				SourcePath: "/mods/alpha/plugin.txt",
				SHA256:     "alpha-hash",
				SizeBytes:  5,
			},
			betaID: {
				SourcePath: "/mods/beta/plugin.txt",
				SHA256:     "beta-hash",
				SizeBytes:  4,
			},
		},
		Writers: []deployment.WriterEntry{
			{
				SourceKind: deployment.SourceKindMod,
				ModID:      &alphaID,
				ModName:    "Alpha",
				LoadOrder:  0,
				IsWinner:   false,
				WouldWrite: true,
			},
			{
				SourceKind: deployment.SourceKindMod,
				ModID:      &betaID,
				ModName:    "Beta",
				LoadOrder:  1,
				IsWinner:   true,
				WouldWrite: false,
			},
		},
	}

	applied := rules.ApplyPerFileWinner(&file, rules.DeploymentRule{
		RuleKind:    rules.RuleKindPerFileWinner,
		WinnerModID: alphaID,
	})
	if !applied {
		t.Fatal("ApplyPerFileWinner() = false, want true")
	}
	if file.SHA256 != "alpha-hash" {
		t.Fatalf("SHA256 = %q, want alpha-hash", file.SHA256)
	}
	if !file.Writers[0].IsWinner || !file.Writers[1].WouldWrite {
		t.Fatalf("writers = %+v, want Alpha winner", file.Writers)
	}
	if file.PerFileRuleModID == nil || *file.PerFileRuleModID != alphaID {
		t.Fatalf("PerFileRuleModID = %+v, want alpha mod ID", file.PerFileRuleModID)
	}
}

func TestApplyPerFileWinnerIgnoresUnknownMod(t *testing.T) {
	t.Parallel()

	modID := int64(1)
	file := deployment.DesiredFile{
		ModContentByID: map[int64]deployment.ModFileContent{
			modID: {SHA256: "hash"},
		},
		Writers: []deployment.WriterEntry{
			{
				SourceKind: deployment.SourceKindMod,
				ModID:      &modID,
				ModName:    "Alpha",
			},
		},
	}

	applied := rules.ApplyPerFileWinner(&file, rules.DeploymentRule{
		RuleKind:    rules.RuleKindPerFileWinner,
		WinnerModID: 99,
	})
	if applied {
		t.Fatal("ApplyPerFileWinner() = true, want false for unknown mod")
	}
}

func TestParseSetPerFileWinnerAction(t *testing.T) {
	t.Parallel()

	modID, ok := rules.ParseSetPerFileWinnerAction(rules.FormatSetPerFileWinnerAction(42))
	if !ok || modID != 42 {
		t.Fatalf("ParseSetPerFileWinnerAction() = (%d, %v), want (42, true)", modID, ok)
	}

	if _, ok := rules.ParseSetPerFileWinnerAction("clear_conflict_rule"); ok {
		t.Fatal("ParseSetPerFileWinnerAction(clear) = true, want false")
	}
}
