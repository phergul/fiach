package review

import (
	"time"

	"github.com/phergul/fiach/internal/deployment"
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
	AppliedAt     *time.Time
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
		PlanMode:      string(entry.Plan.Mode),
		StatusCounts:  statusCounts,
		CanApply:      entry.Plan.CanApply(),
		BlockingCount: entry.Plan.BlockingCount(),
		WarningCount:  entry.Plan.WarningCount(),
		AppliedAt:     entry.AppliedAt,
	}
}

func StatusPriority(status deployment.FileStatus) int {
	switch status {
	case deployment.FileStatusBlocked:
		return 6
	case deployment.FileStatusDrifted:
		return 5
	case deployment.FileStatusConflict:
		return 4
	case deployment.FileStatusExternal:
		return 3
	case deployment.FileStatusSkipped:
		return 3
	case deployment.FileStatusDeleted:
		return 3
	case deployment.FileStatusRestored:
		return 2
	case deployment.FileStatusReplaced:
		return 2
	case deployment.FileStatusAdded:
		return 1
	default:
		return 0
	}
}

func RollUpStatus(current deployment.FileStatus, candidate deployment.FileStatus) deployment.FileStatus {
	if StatusPriority(candidate) > StatusPriority(current) {
		return candidate
	}
	return current
}
