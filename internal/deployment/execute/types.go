package execute

import (
	"context"
	"time"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
)

const journalVersion = 1

type Context struct {
	GameID             int64
	ProfileID          int64
	GameInstallPath    string
	GameModStoragePath string
	PreviewHash        string
	PlanMode           planner.PlanMode
	Plan               planner.DeploymentPlan
	Desired            deployment.DesiredState
	AppliedFileStates  []appliedstate.PersistedFileState
	FirstApplyOutcome  FirstApplyOutcome
	Now                func() time.Time
}

type AppliedStateSaver interface {
	SaveIncrementalAppliedProfileState(
		ctx context.Context,
		gameID int64,
		profileID int64,
		installPath string,
		plan planner.DeploymentPlan,
		desired deployment.DesiredState,
		existingStates []appliedstate.PersistedFileState,
	) error
	SaveFirstApplyAppliedProfileState(
		ctx context.Context,
		gameID int64,
		profileID int64,
		installPath string,
		plan planner.DeploymentPlan,
		desired deployment.DesiredState,
		outcome FirstApplyOutcome,
		previewHash string,
	) error
}

type Result struct {
	Success        bool
	CompletedCount int
	SkippedCount   int
	Message        string
	RolledBack     bool
}
