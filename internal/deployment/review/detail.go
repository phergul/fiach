package review

import (
	"fmt"
	"sort"
	"time"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/storage"
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
	DriftKind        deployment.DriftKind
	DriftExplanation string
	Comparison       StateComparison
	LastAppliedAt    *time.Time
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

	desiredFile, hasDesired := entry.Desired.Files[canonicalPath]

	comparison := buildStateComparison(pathPlan.Applied, pathPlan.Current, pathPlan.Desired)

	var lastAppliedAt *time.Time
	if pathPlan.LastAppliedAt != "" {
		if parsed, ok := storage.ParseAppliedTimestamp(pathPlan.LastAppliedAt); ok {
			lastAppliedAt = &parsed
		}
	}

	relativePathValue := pathPlan.GameRelativePath
	if relativePathValue == "" && hasDesired {
		relativePathValue = desiredFile.GameRelativePath
	}

	var writerStack []deployment.WriterEntry
	var explanation string
	var conflictCategory deployment.ConflictCategory
	if hasDesired {
		writerStack = append([]deployment.WriterEntry(nil), desiredFile.Writers...)
		explanation = desiredFile.Explanation
		conflictCategory = pathPlan.ConflictCategory
		if conflictCategory == "" {
			conflictCategory = desiredFile.ConflictCategory
		}
	}

	return FileDetail{
		RelativePath: relativePathValue,
		States: FourStateView{
			Baseline: toFileStateView(pathPlan.Baseline),
			Applied:  toFileStateView(pathPlan.Applied),
			Current:  toFileStateView(pathPlan.Current),
			Desired:  toFileStateView(pathPlan.Desired),
		},
		WriterStack:      writerStack,
		ConflictCategory: conflictCategory,
		FileStatus:       pathPlan.FileStatus,
		PlannedAction:    pathPlan.PlannedAction,
		RiskLevel:        pathPlan.RiskLevel,
		Explanation:      explanation,
		BackupAvailable:  pathPlan.BaselineBackupPath != "",
		AvailableActions: nil,
		DriftKind:        pathPlan.DriftKind,
		LastAppliedAt:    lastAppliedAt,
		DriftExplanation: buildDriftExplanation(pathPlan.DriftKind, comparison, pathPlan.FileStatus),
		Comparison:       comparison,
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
		desiredSHA256 := ""
		if desiredFile, found := entry.Desired.Files[canonicalPath]; found {
			desiredSHA256 = desiredFile.SHA256
		} else if pathPlan.Desired.Exists {
			desiredSHA256 = pathPlan.Desired.SHA256
		}
		input.Paths = append(input.Paths, previewHashPath{
			Path:          canonicalPath,
			DesiredSHA256: desiredSHA256,
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
