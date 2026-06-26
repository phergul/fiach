package planner

import "github.com/phergul/fiach/internal/deployment"

type PlanMode string

const (
	PlanModeFirstApply PlanMode = "first_apply"
)

type ReapplyAction string

const (
	ReapplyCreate  ReapplyAction = "create"
	ReapplyReplace ReapplyAction = "replace"
	ReapplyBlock   ReapplyAction = "block"
)

type FileStateSnapshot struct {
	Exists    bool
	SHA256    string
	SizeBytes int64
	Label     string
}

type PathPlan struct {
	GameRelativePath string
	PlannedAction    ReapplyAction
	FileStatus       deployment.FileStatus
	RiskLevel        deployment.RiskLevel
	ConflictCategory deployment.ConflictCategory
	Current          FileStateSnapshot
	Desired          FileStateSnapshot
}

type FirstApplyPlan struct {
	Mode   PlanMode
	Paths  map[string]PathPlan
	Issues []deployment.PlanIssue
}

func (p FirstApplyPlan) CanApply() bool {
	for _, issue := range p.Issues {
		if issue.Severity == deployment.PlanIssueSeverityError {
			return false
		}
	}
	for _, pathPlan := range p.Paths {
		if pathPlan.PlannedAction == ReapplyBlock {
			return false
		}
	}
	return true
}

func (p FirstApplyPlan) BlockingCount() int {
	count := 0
	for _, pathPlan := range p.Paths {
		if pathPlan.PlannedAction == ReapplyBlock {
			count++
		}
	}
	return count
}

func (p FirstApplyPlan) WarningCount() int {
	count := 0
	for _, issue := range p.Issues {
		if issue.Severity == deployment.PlanIssueSeverityWarning {
			count++
		}
	}
	return count
}
