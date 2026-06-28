package review

import (
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/drift"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/deployment/rules"
)

func BuildAvailableDriftActions(pathPlan planner.PathPlan, hasDesired bool) []string {
	return buildAvailableDriftActions(pathPlan, hasDesired)
}

func BuildAvailableConflictActions(
	desiredFile deployment.DesiredFile,
	conflictCategory deployment.ConflictCategory,
	savedRule *rules.DeploymentRule,
) []string {
	modWriters := conflictModWriters(desiredFile.Writers)
	if len(modWriters) < 2 {
		return nil
	}

	if conflictCategory != deployment.ConflictExpectedOverwrite &&
		conflictCategory != deployment.ConflictAmbiguousOverwrite &&
		savedRule == nil {
		return nil
	}

	actions := make([]string, 0, len(modWriters)+1)
	for _, writer := range modWriters {
		if writer.ModID == nil {
			continue
		}
		actions = append(actions, rules.FormatSetPerFileWinnerAction(*writer.ModID))
	}
	if savedRule != nil {
		actions = append(actions, rules.ActionClearConflictRule)
	}

	return actions
}

func SavedConflictRuleModName(desiredFile deployment.DesiredFile, savedRule *rules.DeploymentRule) string {
	if savedRule == nil {
		return ""
	}

	for _, writer := range desiredFile.Writers {
		if writer.SourceKind != deployment.SourceKindMod || writer.ModID == nil {
			continue
		}
		if *writer.ModID == savedRule.WinnerModID {
			return writer.ModName
		}
	}

	return ""
}

func conflictModWriters(writers []deployment.WriterEntry) []deployment.WriterEntry {
	result := make([]deployment.WriterEntry, 0, len(writers))
	for _, writer := range writers {
		if writer.SourceKind == deployment.SourceKindMod {
			result = append(result, writer)
		}
	}
	return result
}

func buildAvailableDriftActions(pathPlan planner.PathPlan, hasDesired bool) []string {
	if persistedDecision := persistedDecisionValue(pathPlan.UserDecision); persistedDecision != "" {
		if pathPlan.PlannedAction == planner.ReapplyRequireDecision {
			return availableActionsForUnresolvedDrift(pathPlan, hasDesired)
		}

		return []string{drift.UserDecisionClear}
	}

	if pathPlan.PlannedAction != planner.ReapplyRequireDecision {
		return nil
	}

	return availableActionsForUnresolvedDrift(pathPlan, hasDesired)
}

func availableActionsForUnresolvedDrift(pathPlan planner.PathPlan, hasDesired bool) []string {
	switch pathPlan.DriftKind {
	case deployment.DriftMissing:
		return []string{
			drift.UserDecisionSkipped,
			drift.UserDecisionBackupAndApply,
		}
	default:
		if !hasDesired {
			return []string{
				drift.UserDecisionBackupAndApply,
				drift.UserDecisionKeepExternal,
				drift.UserDecisionSkipped,
			}
		}

		return []string{
			drift.UserDecisionBackupAndApply,
			drift.UserDecisionKeepExternal,
			drift.UserDecisionSkipped,
		}
	}
}

func persistedDecisionValue(decision *string) string {
	if decision == nil {
		return ""
	}

	value := *decision
	if drift.IsPersistedDecision(value) {
		return value
	}

	return ""
}

func buildUserDecisionLabel(pathPlan planner.PathPlan) string {
	return drift.DecisionLabel(persistedDecisionValue(pathPlan.UserDecision))
}
