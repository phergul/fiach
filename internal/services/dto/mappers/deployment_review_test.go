package mappers_test

import (
	"testing"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/deployment/review"
	"github.com/phergul/fiach/internal/services/dto/mappers"
)

func TestToDTODeploymentFileDetailIncludesNullFourStateSlots(t *testing.T) {
	t.Parallel()

	detail := review.FileDetail{
		RelativePath: "Data/SkyUI.esp",
		States: review.FourStateView{
			Baseline: review.FileStateView{Exists: false},
			Applied:  review.FileStateView{Exists: false},
			Current:  review.FileStateView{Exists: false},
			Desired: review.FileStateView{
				Exists:    true,
				SHA256:    "desired",
				SizeBytes: 7,
				Label:     "Desired content",
			},
		},
		WriterStack: []deployment.WriterEntry{
			{
				Order:      1,
				SourceKind: deployment.SourceKindMod,
				SourceID:   "mod:1",
				ModName:    "SkyUI",
				LoadOrder:  0,
				IsWinner:   true,
				WouldWrite: true,
			},
		},
		FileStatus:    deployment.FileStatusAdded,
		PlannedAction: planner.ReapplyCreate,
		RiskLevel:     deployment.RiskNone,
		Explanation:   "Will add file.",
	}

	dtoDetail := mappers.ToDTODeploymentFileDetail(detail, 42)

	if dtoDetail.States.Baseline == nil || dtoDetail.States.Applied == nil || dtoDetail.States.Current == nil || dtoDetail.States.Desired == nil {
		t.Fatalf("ToDTODeploymentFileDetail() states = %+v, want all four slots populated", dtoDetail.States)
	}
	if dtoDetail.States.Baseline.Exists || dtoDetail.States.Applied.Exists {
		t.Fatalf("baseline/applied = %+v / %+v, want Exists=false", dtoDetail.States.Baseline, dtoDetail.States.Applied)
	}
	if !dtoDetail.States.Desired.Exists || dtoDetail.States.Desired.SHA256 != "desired" {
		t.Fatalf("desired = %+v, want populated desired state", dtoDetail.States.Desired)
	}
	if len(dtoDetail.WriterStack) != 1 || dtoDetail.WriterStack[0].ModName != "SkyUI" {
		t.Fatalf("writer stack = %+v, want SkyUI winner", dtoDetail.WriterStack)
	}
	if dtoDetail.WriterStack[0].DisplayLoadOrder != 1 {
		t.Fatalf("DisplayLoadOrder = %d, want 1", dtoDetail.WriterStack[0].DisplayLoadOrder)
	}
}
