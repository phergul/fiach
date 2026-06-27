package review

import (
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
)

type Summary struct {
	GameID        int64
	ProfileID     int64
	ProfileName   string
	PlanMode      string
	StatusCounts  map[string]int
	CanApply      bool
	BlockingCount int
	WarningCount  int
}

func BuildSummary(entry CachedPreview) Summary {
	statusCounts := map[string]int{}
	for _, pathPlan := range entry.Plan.Paths {
		statusCounts[string(pathPlan.FileStatus)]++
	}

	return Summary{
		GameID:        entry.GameID,
		ProfileID:     entry.ProfileID,
		ProfileName:   entry.ProfileName,
		PlanMode:      string(planner.PlanModeFirstApply),
		StatusCounts:  statusCounts,
		CanApply:      entry.Plan.CanApply(),
		BlockingCount: entry.Plan.BlockingCount(),
		WarningCount:  entry.Plan.WarningCount(),
	}
}

func StatusPriority(status deployment.FileStatus) int {
	switch status {
	case deployment.FileStatusBlocked:
		return 5
	case deployment.FileStatusConflict:
		return 4
	case deployment.FileStatusReplaced:
		return 3
	case deployment.FileStatusAdded:
		return 2
	default:
		return 1
	}
}

func RollUpStatus(current deployment.FileStatus, candidate deployment.FileStatus) deployment.FileStatus {
	if StatusPriority(candidate) > StatusPriority(current) {
		return candidate
	}
	return current
}
