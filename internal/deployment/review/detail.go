package review

import (
	"fmt"
	"sort"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/fileops"
)

type FileDetail struct {
	RelativePath     string
	States           FourStateView
	WriterStack      []deployment.WriterEntry
	ConflictCategory deployment.ConflictCategory
	FileStatus       deployment.FileStatus
	PlannedAction    planner.ReapplyAction
	RiskLevel        deployment.RiskLevel
	Explanation      string
	BackupAvailable  bool
	AvailableActions []string
}

type FourStateView struct {
	Baseline FileStateView
	Applied  FileStateView
	Current  FileStateView
	Desired  FileStateView
}

type FileStateView struct {
	Exists    bool
	SHA256    string
	SizeBytes int64
	Label     string
}

type previewHashInput struct {
	ProfileID int64
	GameID    int64
	PlanMode  string
	Paths     []previewHashPath
}

type previewHashPath struct {
	Path          string
	DesiredSHA256 string
	AppliedSHA256 string
	CurrentSHA256 string
	DriftKind     string
	PlannedAction string
	FileStatus    string
}

func BuildFileDetail(entry CachedPreview, relativePath string) (FileDetail, error) {
	canonicalPath := deployment.CanonicalGameRelativePath(relativePath)
	pathPlan, found := entry.Plan.Paths[canonicalPath]
	if !found {
		return FileDetail{}, fmt.Errorf("deployment path %q was not found in preview", relativePath)
	}

	desiredFile, found := entry.Desired.Files[canonicalPath]
	if !found {
		return FileDetail{}, fmt.Errorf("desired file %q was not found in preview", relativePath)
	}

	return FileDetail{
		RelativePath: desiredFile.GameRelativePath,
		States: FourStateView{
			Baseline: toFileStateView(pathPlan.Baseline),
			Applied:  toFileStateView(pathPlan.Applied),
			Current:  toFileStateView(pathPlan.Current),
			Desired:  toFileStateView(pathPlan.Desired),
		},
		WriterStack:      append([]deployment.WriterEntry(nil), desiredFile.Writers...),
		ConflictCategory: pathPlan.ConflictCategory,
		FileStatus:       pathPlan.FileStatus,
		PlannedAction:    pathPlan.PlannedAction,
		RiskLevel:        pathPlan.RiskLevel,
		Explanation:      desiredFile.Explanation,
		BackupAvailable:  pathPlan.BaselineBackupPath != "",
		AvailableActions: nil,
	}, nil
}

func PreviewHash(entry CachedPreview) (string, error) {
	input := previewHashInput{
		ProfileID: entry.ProfileID,
		GameID:    entry.GameID,
		PlanMode:  string(entry.Plan.Mode),
	}

	canonicalPaths := make([]string, 0, len(entry.Plan.Paths))
	for canonicalPath := range entry.Plan.Paths {
		canonicalPaths = append(canonicalPaths, canonicalPath)
	}
	sort.Strings(canonicalPaths)

	input.Paths = make([]previewHashPath, 0, len(canonicalPaths))
	for _, canonicalPath := range canonicalPaths {
		pathPlan := entry.Plan.Paths[canonicalPath]
		desiredFile := entry.Desired.Files[canonicalPath]
		input.Paths = append(input.Paths, previewHashPath{
			Path:          canonicalPath,
			DesiredSHA256: desiredFile.SHA256,
			AppliedSHA256: pathPlan.Applied.SHA256,
			CurrentSHA256: pathPlan.Current.SHA256,
			DriftKind:     string(pathPlan.DriftKind),
			PlannedAction: string(pathPlan.PlannedAction),
			FileStatus:    string(pathPlan.FileStatus),
		})
	}

	return fileops.HashJSON(input)
}

func toFileStateView(snapshot planner.FileStateSnapshot) FileStateView {
	return FileStateView{
		Exists:    snapshot.Exists,
		SHA256:    snapshot.SHA256,
		SizeBytes: snapshot.SizeBytes,
		Label:     snapshot.Label,
	}
}
