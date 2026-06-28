package dto

import "time"

type DeploymentSummary struct {
	GameID          int64
	ProfileID       int64
	ProfileName     string
	AppliedAt       *time.Time
	PlanMode        string
	StatusCounts    map[string]int
	CanApply        bool
	PreviewHash     string
	BlockingCount   int
	WarningCount    int
	PreviousApplyAt *time.Time
}

type DeploymentTreeNode struct {
	Path          string
	Name          string
	IsDirectory   bool
	Status        string
	PlannedAction string
	RiskLevel     string
	ChildCount    int
	HasChildren   bool
	Children      []DeploymentTreeNode
}

type FileStateView struct {
	Exists    bool
	SHA256    string
	SizeBytes int64
	Label     string
}

type FourStateView struct {
	Baseline *FileStateView
	Applied  *FileStateView
	Current  *FileStateView
	Desired  *FileStateView
}

type WriterEntryDTO struct {
	Order            int
	SourceKind       string
	SourceID         string
	ModID            *int64
	ModName          string
	LoadOrder        int64
	DisplayLoadOrder int64
	IsWinner         bool
	WouldWrite       bool
}

type StateComparison struct {
	AppliedMatchesCurrent bool
	AppliedMatchesDesired bool
	CurrentMatchesDesired bool
}

type DeploymentFileDetail struct {
	RelativePath             string
	States                   FourStateView
	WriterStack              []WriterEntryDTO
	ConflictCategory         string
	FileStatus               string
	PlannedAction            string
	RiskLevel                string
	Explanation              string
	BackupAvailable          bool
	AvailableActions         []string
	ConflictAvailableActions []string
	SavedConflictRuleModID   *int64
	SavedConflictRuleModName string
	ProfileModsURL           string
	UserDecision             string
	UserDecisionLabel        string
	DriftKind                string
	DriftExplanation         string
	Comparison               StateComparison
	LastAppliedAt            *time.Time
}

type DeploymentReviewPreview struct {
	Summary     DeploymentSummary
	Root        DeploymentTreeNode
	PreviewHash string
}
