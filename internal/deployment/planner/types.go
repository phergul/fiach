package planner

import "github.com/phergul/fiach/internal/deployment"

type PlanMode string

const (
	PlanModeFirstApply  PlanMode = "first_apply"
	PlanModeIncremental PlanMode = "incremental"
)

type ReapplyAction string

const (
	ReapplyNoOp            ReapplyAction = "noop"
	ReapplyCreate          ReapplyAction = "create"
	ReapplyReplace         ReapplyAction = "replace"
	ReapplyRequireDecision ReapplyAction = "require_decision"
	ReapplyBlock           ReapplyAction = "block"
)

type FileStateSnapshot struct {
	Exists    bool
	SHA256    string
	SizeBytes int64
	Label     string
}

type PathPlan struct {
	GameRelativePath   string
	PlannedAction      ReapplyAction
	FileStatus         deployment.FileStatus
	RiskLevel          deployment.RiskLevel
	ConflictCategory   deployment.ConflictCategory
	DriftKind          deployment.DriftKind
	Baseline           FileStateSnapshot
	Applied            FileStateSnapshot
	Current            FileStateSnapshot
	Desired            FileStateSnapshot
	BaselineBackupPath string
}

type DeploymentPlan struct {
	Mode        PlanMode
	Paths       map[string]PathPlan
	Issues      []deployment.PlanIssue
	PreviewOnly bool
}

func (p DeploymentPlan) CanApply() bool {
	if p.PreviewOnly {
		return false
	}

	for _, issue := range p.Issues {
		if issue.Severity == deployment.PlanIssueSeverityError {
			return false
		}
	}
	for _, pathPlan := range p.Paths {
		if pathPlan.PlannedAction == ReapplyBlock || pathPlan.PlannedAction == ReapplyRequireDecision {
			return false
		}
	}
	return true
}

func (p DeploymentPlan) BlockingCount() int {
	count := 0
	for _, pathPlan := range p.Paths {
		if pathPlan.PlannedAction == ReapplyBlock || pathPlan.PlannedAction == ReapplyRequireDecision {
			count++
		}
	}
	return count
}

func (p DeploymentPlan) WarningCount() int {
	count := 0
	for _, issue := range p.Issues {
		if issue.Severity == deployment.PlanIssueSeverityWarning {
			count++
		}
	}
	return count
}
