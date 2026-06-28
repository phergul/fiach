package mappers

import (
	"maps"

	"fmt"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/execute"
	"github.com/phergul/fiach/internal/deployment/review"
	"github.com/phergul/fiach/internal/loadorder"
	"github.com/phergul/fiach/internal/services/dto"
)

func ToDTODeploymentReviewPreview(entry review.CachedPreview, rootNodes []review.TreeNode) dto.DeploymentReviewPreview {
	summary := review.BuildSummary(entry)
	root := dto.DeploymentTreeNode{
		Path:        "",
		Name:        "",
		IsDirectory: true,
		HasChildren: len(rootNodes) > 0,
		ChildCount:  len(rootNodes),
		Children:    ToDTODeploymentTreeNodes(rootNodes),
	}

	return dto.DeploymentReviewPreview{
		Summary:     ToDTODeploymentSummary(summary, entry.PreviewHash),
		Root:        root,
		PreviewHash: entry.PreviewHash,
	}
}

func ToDTODeploymentSummary(summary review.Summary, previewHash string) dto.DeploymentSummary {
	statusCounts := map[string]int{}
	maps.Copy(statusCounts, summary.StatusCounts)

	return dto.DeploymentSummary{
		GameID:          summary.GameID,
		ProfileID:       summary.ProfileID,
		ProfileName:     summary.ProfileName,
		AppliedAt:       summary.AppliedAt,
		PlanMode:        summary.PlanMode,
		StatusCounts:    statusCounts,
		CanApply:        summary.CanApply,
		PreviewHash:     previewHash,
		BlockingCount:   summary.BlockingCount,
		WarningCount:    summary.WarningCount,
		PreviousApplyAt: summary.AppliedAt,
	}
}

func ToDTODeploymentTreeNodes(nodes []review.TreeNode) []dto.DeploymentTreeNode {
	result := make([]dto.DeploymentTreeNode, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, ToDTODeploymentTreeNode(node))
	}
	return result
}

func ToDTODeploymentTreeNode(node review.TreeNode) dto.DeploymentTreeNode {
	return dto.DeploymentTreeNode{
		Path:          node.Path,
		Name:          node.Name,
		IsDirectory:   node.IsDirectory,
		Status:        string(node.Status),
		PlannedAction: string(node.PlannedAction),
		RiskLevel:     string(node.RiskLevel),
		ChildCount:    node.ChildCount,
		HasChildren:   node.HasChildren,
		Children:      nil,
	}
}

func ToDTODeploymentFileDetail(detail review.FileDetail, gameID int64) dto.DeploymentFileDetail {
	writerStack := make([]dto.WriterEntryDTO, 0, len(detail.WriterStack))
	for _, writer := range detail.WriterStack {
		writerStack = append(writerStack, ToDTOWriterEntryDTO(writer))
	}

	availableActions := append([]string(nil), detail.AvailableActions...)
	conflictAvailableActions := append([]string(nil), detail.ConflictAvailableActions...)

	profileModsURL := ""
	if gameID > 0 {
		profileModsURL = fmt.Sprintf("/library/%d", gameID)
	}

	return dto.DeploymentFileDetail{
		RelativePath:             detail.RelativePath,
		States:                   ToDTOFourStateView(detail.States),
		WriterStack:              writerStack,
		ConflictCategory:         string(detail.ConflictCategory),
		FileStatus:               string(detail.FileStatus),
		PlannedAction:            string(detail.PlannedAction),
		RiskLevel:                string(detail.RiskLevel),
		Explanation:              detail.Explanation,
		BackupAvailable:          detail.BackupAvailable,
		AvailableActions:         availableActions,
		ConflictAvailableActions: conflictAvailableActions,
		SavedConflictRuleModID:   detail.SavedConflictRuleModID,
		SavedConflictRuleModName: detail.SavedConflictRuleModName,
		ProfileModsURL:           profileModsURL,
		UserDecision:             detail.UserDecision,
		UserDecisionLabel:        detail.UserDecisionLabel,
		DriftKind:                string(detail.DriftKind),
		LastAppliedAt:            detail.LastAppliedAt,
		DriftExplanation:         detail.DriftExplanation,
		Comparison: dto.StateComparison{
			AppliedMatchesCurrent: detail.Comparison.AppliedMatchesCurrent,
			AppliedMatchesDesired: detail.Comparison.AppliedMatchesDesired,
			CurrentMatchesDesired: detail.Comparison.CurrentMatchesDesired,
		},
	}
}

func ToDTOWriterEntryDTO(writer deployment.WriterEntry) dto.WriterEntryDTO {
	displayLoadOrder := int64(0)
	if writer.SourceKind == deployment.SourceKindMod {
		displayLoadOrder = loadorder.DisplayIndex(writer.LoadOrder)
	}

	return dto.WriterEntryDTO{
		Order:            writer.Order,
		SourceKind:       string(writer.SourceKind),
		SourceID:         writer.SourceID,
		ModID:            writer.ModID,
		ModName:          writer.ModName,
		LoadOrder:        writer.LoadOrder,
		DisplayLoadOrder: displayLoadOrder,
		IsWinner:         writer.IsWinner,
		WouldWrite:       writer.WouldWrite,
	}
}

func ToDTOFourStateView(states review.FourStateView) dto.FourStateView {
	return dto.FourStateView{
		Baseline: toDTOOptionalFileStateView(states.Baseline),
		Applied:  toDTOOptionalFileStateView(states.Applied),
		Current:  toDTOOptionalFileStateView(states.Current),
		Desired:  toDTOOptionalFileStateView(states.Desired),
	}
}

func toDTOOptionalFileStateView(state review.FileStateView) *dto.FileStateView {
	if !state.Exists && state.SHA256 == "" && state.SizeBytes == 0 && state.Label == "" {
		return &dto.FileStateView{Exists: false}
	}

	return &dto.FileStateView{
		Exists:    state.Exists,
		SHA256:    state.SHA256,
		SizeBytes: state.SizeBytes,
		Label:     state.Label,
	}
}

func ToDTOApplyIncrementalDeploymentResult(result execute.Result) dto.ApplyIncrementalDeploymentResult {
	return dto.ApplyIncrementalDeploymentResult{
		Success:        result.Success,
		CompletedCount: result.CompletedCount,
		SkippedCount:   result.SkippedCount,
		Message:        result.Message,
		RolledBack:     result.RolledBack,
	}
}
