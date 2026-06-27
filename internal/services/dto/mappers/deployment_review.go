package mappers

import (
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/review"
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
	for key, value := range summary.StatusCounts {
		statusCounts[key] = value
	}

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

func ToDTODeploymentFileDetail(detail review.FileDetail) dto.DeploymentFileDetail {
	writerStack := make([]dto.WriterEntryDTO, 0, len(detail.WriterStack))
	for _, writer := range detail.WriterStack {
		writerStack = append(writerStack, ToDTOWriterEntryDTO(writer))
	}

	availableActions := append([]string(nil), detail.AvailableActions...)

	return dto.DeploymentFileDetail{
		RelativePath:     detail.RelativePath,
		States:           ToDTOFourStateView(detail.States),
		WriterStack:      writerStack,
		ConflictCategory: string(detail.ConflictCategory),
		FileStatus:       string(detail.FileStatus),
		PlannedAction:    string(detail.PlannedAction),
		RiskLevel:        string(detail.RiskLevel),
		Explanation:      detail.Explanation,
		BackupAvailable:  detail.BackupAvailable,
		AvailableActions: availableActions,
	}
}

func ToDTOWriterEntryDTO(writer deployment.WriterEntry) dto.WriterEntryDTO {
	return dto.WriterEntryDTO{
		Order:      writer.Order,
		SourceKind: string(writer.SourceKind),
		SourceID:   writer.SourceID,
		ModID:      writer.ModID,
		ModName:    writer.ModName,
		LoadOrder:  writer.LoadOrder,
		IsWinner:   writer.IsWinner,
		WouldWrite: writer.WouldWrite,
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
