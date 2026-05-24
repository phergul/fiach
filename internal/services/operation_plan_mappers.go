package services

import (
	"github.com/phergul/mod-manager/internal/operationplan"
	"github.com/phergul/mod-manager/internal/services/dto"
)

func toDTOOperationPlan(plan operationplan.OperationPlan) dto.OperationPlan {
	return dto.OperationPlan{
		Operations: toDTOOperations(plan.Operations),
		Issues:     toDTOPlanIssues(plan.Issues),
		CanApply:   plan.CanApply,
	}
}

func toInternalOperationPlan(plan dto.OperationPlan) operationplan.OperationPlan {
	return operationplan.OperationPlan{
		Operations: toInternalOperations(plan.Operations),
		Issues:     toInternalPlanIssues(plan.Issues),
		CanApply:   plan.CanApply,
	}
}

func toDTOOperations(operations []operationplan.Operation) []dto.Operation {
	result := make([]dto.Operation, 0, len(operations))
	for _, operation := range operations {
		result = append(result, toDTOOperation(operation))
	}
	return result
}

func toInternalOperations(operations []dto.Operation) []operationplan.Operation {
	result := make([]operationplan.Operation, 0, len(operations))
	for _, operation := range operations {
		result = append(result, toInternalOperation(operation))
	}
	return result
}

func toDTOOperation(operation operationplan.Operation) dto.Operation {
	return dto.Operation{
		Type:       dto.OperationType(operation.Type),
		SourcePath: operation.SourcePath,
		TargetPath: operation.TargetPath,
		BackupPath: operation.BackupPath,
		Conflict:   operation.Conflict,
		Mod:        toDTOModContext(operation.Mod),
	}
}

func toInternalOperation(operation dto.Operation) operationplan.Operation {
	return operationplan.Operation{
		Type:       operationplan.OperationType(operation.Type),
		SourcePath: operation.SourcePath,
		TargetPath: operation.TargetPath,
		BackupPath: operation.BackupPath,
		Conflict:   operation.Conflict,
		Mod:        toInternalModContext(operation.Mod),
	}
}

func toDTOModContext(mod operationplan.ModContext) dto.ModContext {
	return dto.ModContext(mod)
}

func toInternalModContext(mod dto.ModContext) operationplan.ModContext {
	return operationplan.ModContext(mod)
}

func toDTOModContextPtr(mod *operationplan.ModContext) *dto.ModContext {
	if mod == nil {
		return nil
	}
	result := toDTOModContext(*mod)
	return &result
}

func toInternalModContextPtr(mod *dto.ModContext) *operationplan.ModContext {
	if mod == nil {
		return nil
	}
	result := toInternalModContext(*mod)
	return &result
}

func toDTOPlanIssues(issues []operationplan.PlanIssue) []dto.PlanIssue {
	result := make([]dto.PlanIssue, 0, len(issues))
	for _, issue := range issues {
		result = append(result, toDTOPlanIssue(issue))
	}
	return result
}

func toInternalPlanIssues(issues []dto.PlanIssue) []operationplan.PlanIssue {
	result := make([]operationplan.PlanIssue, 0, len(issues))
	for _, issue := range issues {
		result = append(result, toInternalPlanIssue(issue))
	}
	return result
}

func toDTOPlanIssue(issue operationplan.PlanIssue) dto.PlanIssue {
	return dto.PlanIssue{
		Severity:   dto.PlanIssueSeverity(issue.Severity),
		Kind:       dto.PlanIssueKind(issue.Kind),
		Message:    issue.Message,
		ProfileID:  issue.ProfileID,
		SourcePath: issue.SourcePath,
		TargetPath: issue.TargetPath,
		Mod:        toDTOModContextPtr(issue.Mod),
	}
}

func toInternalPlanIssue(issue dto.PlanIssue) operationplan.PlanIssue {
	return operationplan.PlanIssue{
		Severity:   operationplan.PlanIssueSeverity(issue.Severity),
		Kind:       operationplan.PlanIssueKind(issue.Kind),
		Message:    issue.Message,
		ProfileID:  issue.ProfileID,
		SourcePath: issue.SourcePath,
		TargetPath: issue.TargetPath,
		Mod:        toInternalModContextPtr(issue.Mod),
	}
}
