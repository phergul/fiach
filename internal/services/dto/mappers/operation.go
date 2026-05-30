package mappers

import (
	"github.com/phergul/fiach/internal/operationplan"
	"github.com/phergul/fiach/internal/services/dto"
)

func ToDTOOperationPlan(plan operationplan.OperationPlan) dto.OperationPlan {
	return dto.OperationPlan{
		Operations: ToDTOOperations(plan.Operations),
		Issues:     ToDTOPlanIssues(plan.Issues),
		CanApply:   plan.CanApply,
	}
}

func ToInternalOperationPlan(plan dto.OperationPlan) operationplan.OperationPlan {
	return operationplan.OperationPlan{
		Operations: ToInternalOperations(plan.Operations),
		Issues:     ToInternalPlanIssues(plan.Issues),
		CanApply:   plan.CanApply,
	}
}

func ToDTOOperations(operations []operationplan.Operation) []dto.Operation {
	result := make([]dto.Operation, 0, len(operations))
	for _, operation := range operations {
		result = append(result, ToDTOOperation(operation))
	}
	return result
}

func ToInternalOperations(operations []dto.Operation) []operationplan.Operation {
	result := make([]operationplan.Operation, 0, len(operations))
	for _, operation := range operations {
		result = append(result, ToInternalOperation(operation))
	}
	return result
}

func ToDTOOperation(operation operationplan.Operation) dto.Operation {
	return dto.Operation{
		Type:       dto.OperationType(operation.Type),
		SourcePath: operation.SourcePath,
		TargetPath: operation.TargetPath,
		BackupPath: operation.BackupPath,
		Conflict:   operation.Conflict,
		Mod:        ToDTOModContext(operation.Mod),
	}
}

func ToInternalOperation(operation dto.Operation) operationplan.Operation {
	return operationplan.Operation{
		Type:       operationplan.OperationType(operation.Type),
		SourcePath: operation.SourcePath,
		TargetPath: operation.TargetPath,
		BackupPath: operation.BackupPath,
		Conflict:   operation.Conflict,
		Mod:        ToInternalModContext(operation.Mod),
	}
}

func ToDTOModContext(mod operationplan.ModContext) dto.ModContext {
	return dto.ModContext(mod)
}

func ToInternalModContext(mod dto.ModContext) operationplan.ModContext {
	return operationplan.ModContext(mod)
}

func ToDTOModContextPtr(mod *operationplan.ModContext) *dto.ModContext {
	if mod == nil {
		return nil
	}
	result := ToDTOModContext(*mod)
	return &result
}

func ToInternalModContextPtr(mod *dto.ModContext) *operationplan.ModContext {
	if mod == nil {
		return nil
	}
	result := ToInternalModContext(*mod)
	return &result
}

func ToDTOPlanIssues(issues []operationplan.PlanIssue) []dto.PlanIssue {
	result := make([]dto.PlanIssue, 0, len(issues))
	for _, issue := range issues {
		result = append(result, ToDTOPlanIssue(issue))
	}
	return result
}

func ToInternalPlanIssues(issues []dto.PlanIssue) []operationplan.PlanIssue {
	result := make([]operationplan.PlanIssue, 0, len(issues))
	for _, issue := range issues {
		result = append(result, ToInternalPlanIssue(issue))
	}
	return result
}

func ToDTOPlanIssue(issue operationplan.PlanIssue) dto.PlanIssue {
	return dto.PlanIssue{
		Severity:                    dto.PlanIssueSeverity(issue.Severity),
		Kind:                        dto.PlanIssueKind(issue.Kind),
		Message:                     issue.Message,
		ProfileID:                   issue.ProfileID,
		SourcePath:                  issue.SourcePath,
		TargetPath:                  issue.TargetPath,
		Mod:                         ToDTOModContextPtr(issue.Mod),
		ConflictingOperationIndexes: append([]int{}, issue.ConflictingOperationIndexes...),
	}
}

func ToInternalPlanIssue(issue dto.PlanIssue) operationplan.PlanIssue {
	return operationplan.PlanIssue{
		Severity:                    operationplan.PlanIssueSeverity(issue.Severity),
		Kind:                        operationplan.PlanIssueKind(issue.Kind),
		Message:                     issue.Message,
		ProfileID:                   issue.ProfileID,
		SourcePath:                  issue.SourcePath,
		TargetPath:                  issue.TargetPath,
		Mod:                         ToInternalModContextPtr(issue.Mod),
		ConflictingOperationIndexes: append([]int{}, issue.ConflictingOperationIndexes...),
	}
}
