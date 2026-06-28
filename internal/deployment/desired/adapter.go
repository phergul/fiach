package desired

import (
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/profile"
)

type DesiredFileMapping struct {
	SourcePath       string
	GameRelativePath string
	SHA256           string
	SizeBytes        int64
}

type DesiredInventoryResult struct {
	Mappings []DesiredFileMapping
	Issues   []deployment.PlanIssue
}

type DesiredFileAdapter interface {
	InventoryFiles(input profile.StrategyBuildInput) (DesiredInventoryResult, error)
}
