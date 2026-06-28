package rules

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/phergul/fiach/internal/deployment"
)

const (
	RuleKindPerFileWinner = "per_file_winner"

	ActionClearConflictRule      = "clear_conflict_rule"
	ActionSetPerFileWinnerPrefix = "set_per_file_winner:"
)

type DeploymentRule struct {
	ProfileID        int64
	GameRelativePath string
	RuleKind         string
	WinnerModID      int64
}

func IndexPerFileWinnerRules(rules []DeploymentRule) map[string]DeploymentRule {
	indexed := make(map[string]DeploymentRule, len(rules))
	for _, rule := range rules {
		if rule.RuleKind != RuleKindPerFileWinner {
			continue
		}
		if rule.WinnerModID <= 0 {
			continue
		}
		canonicalPath := deployment.CanonicalGameRelativePath(rule.GameRelativePath)
		indexed[canonicalPath] = rule
	}

	return indexed
}

func ApplyPerFileWinner(file *deployment.DesiredFile, rule DeploymentRule) bool {
	if file == nil || rule.WinnerModID <= 0 {
		return false
	}

	content, hasContent := file.ModContentByID[rule.WinnerModID]
	if !hasContent {
		return false
	}

	winnerIndex := -1
	for index, writer := range file.Writers {
		if writer.SourceKind != deployment.SourceKindMod || writer.ModID == nil {
			continue
		}
		if *writer.ModID == rule.WinnerModID {
			winnerIndex = index
			break
		}
	}
	if winnerIndex < 0 {
		return false
	}

	for index := range file.Writers {
		file.Writers[index].IsWinner = index == winnerIndex
		if file.Writers[index].SourceKind == deployment.SourceKindMod {
			file.Writers[index].WouldWrite = index != winnerIndex
		}
	}

	file.Winner = file.Writers[winnerIndex]
	file.SourcePath = content.SourcePath
	file.SHA256 = content.SHA256
	file.SizeBytes = content.SizeBytes
	ruleModID := rule.WinnerModID
	file.PerFileRuleModID = &ruleModID

	return true
}

func FormatSetPerFileWinnerAction(modID int64) string {
	return ActionSetPerFileWinnerPrefix + strconv.FormatInt(modID, 10)
}

func ParseSetPerFileWinnerAction(action string) (int64, bool) {
	if !strings.HasPrefix(action, ActionSetPerFileWinnerPrefix) {
		return 0, false
	}

	modID, err := strconv.ParseInt(strings.TrimPrefix(action, ActionSetPerFileWinnerPrefix), 10, 64)
	if err != nil || modID <= 0 {
		return 0, false
	}

	return modID, true
}

func IsClearConflictRuleAction(action string) bool {
	return strings.TrimSpace(action) == ActionClearConflictRule
}

func ValidateConflictAction(action string) error {
	action = strings.TrimSpace(action)
	if action == "" {
		return fmt.Errorf("a conflict action is required")
	}
	if IsClearConflictRuleAction(action) {
		return nil
	}
	if _, ok := ParseSetPerFileWinnerAction(action); ok {
		return nil
	}

	return fmt.Errorf("conflict action %q is not supported", action)
}
