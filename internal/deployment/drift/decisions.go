package drift

const (
	UserDecisionBackupAndApply = "backup_and_apply"
	UserDecisionKeepExternal   = "keep_external"
	UserDecisionSkipped        = "skipped"
	UserDecisionClear          = "clear"
)

func IsPersistedDecision(decision string) bool {
	switch decision {
	case UserDecisionBackupAndApply, UserDecisionKeepExternal, UserDecisionSkipped:
		return true
	default:
		return false
	}
}

func IsClearInput(decision string) bool {
	return decision == UserDecisionClear
}

func DecisionLabel(decision string) string {
	switch decision {
	case UserDecisionBackupAndApply:
		return "Backup and apply"
	case UserDecisionKeepExternal:
		return "Keep external"
	case UserDecisionSkipped:
		return "Skipped"
	default:
		return ""
	}
}

func IsKeepExternalDecision(decision *string) bool {
	if decision == nil {
		return false
	}

	return *decision == UserDecisionKeepExternal
}

func IsSkippedDecision(decision *string) bool {
	if decision == nil {
		return false
	}

	return *decision == UserDecisionSkipped
}

func IsBackupAndApplyDecision(decision *string) bool {
	if decision == nil {
		return false
	}

	return *decision == UserDecisionBackupAndApply
}
