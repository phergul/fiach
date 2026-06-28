package inspect

import (
	"archive/zip"
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/deployment/review"
)

func TestSelectDefaultPairUsesCurrentVsDesired(t *testing.T) {
	t.Parallel()

	pair := SelectDefaultPair(string(planner.PlanModeFirstApply), review.StateComparison{})
	if pair.Left != StateCurrent || pair.Right != StateDesired {
		t.Fatalf("SelectDefaultPair() = %+v, want current vs desired", pair)
	}
}

func TestSelectDefaultPairUsesAppliedVsCurrentWhenDrifted(t *testing.T) {
	t.Parallel()

	pair := SelectDefaultPair(string(planner.PlanModeIncremental), review.StateComparison{
		AppliedMatchesCurrent: false,
	})
	if pair.Left != StateApplied || pair.Right != StateCurrent {
		t.Fatalf("SelectDefaultPair() = %+v, want applied vs current", pair)
	}
}

func TestBuildTextDiffProducesInsertAndDeleteLines(t *testing.T) {
	t.Parallel()

	leftDir := t.TempDir()
	rightDir := t.TempDir()

	leftPath := filepath.Join(leftDir, "config.ini")
	rightPath := filepath.Join(rightDir, "config.ini")

	if err := os.WriteFile(leftPath, []byte("a=1\nb=2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(left) error = %v", err)
	}
	if err := os.WriteFile(rightPath, []byte("a=1\nc=3\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(right) error = %v", err)
	}

	lines, limited, reason, err := buildTextDiff(leftPath, rightPath)
	if err != nil {
		t.Fatalf("buildTextDiff() error = %v", err)
	}
	if limited || reason != "" {
		t.Fatalf("buildTextDiff() limited = %v reason = %q, want full diff", limited, reason)
	}

	hasDelete := false
	hasInsert := false
	for _, line := range lines {
		if line.Kind == "delete" && line.Line == "b=2" {
			hasDelete = true
		}
		if line.Kind == "insert" && line.Line == "c=3" {
			hasInsert = true
		}
	}
	if !hasDelete || !hasInsert {
		t.Fatalf("buildTextDiff() = %+v, want delete and insert lines", lines)
	}
}

func TestReadTextFileRejectsOversizedFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "large.txt")
	if err := os.WriteFile(path, bytes.Repeat([]byte("a"), int(MaxTextDiffBytes)+1), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, limited, reason, err := readTextFile(path)
	if err != nil {
		t.Fatalf("readTextFile() error = %v", err)
	}
	if !limited || reason == "" {
		t.Fatalf("readTextFile() limited = %v reason = %q, want size limit", limited, reason)
	}
}

func TestReadImageMetadata(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "icon.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	metadata, err := readImageMetadata(path)
	if err != nil {
		t.Fatalf("readImageMetadata() error = %v", err)
	}
	if metadata.Format != "png" || metadata.Width != 4 || metadata.Height != 2 {
		t.Fatalf("readImageMetadata() = %+v, want png 4x2", metadata)
	}
}

func TestListArchiveRejectsTraversalEntry(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "mod.zip")
	archiveFile, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	writer := zip.NewWriter(archiveFile)
	entry, err := writer.Create("../escape.txt")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := entry.Write([]byte("bad")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := archiveFile.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	_, err = listArchive(path)
	if err == nil {
		t.Fatal("listArchive() error = nil, want traversal rejection")
	}
}

func TestInspectTextConfigFile(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	modRoot := t.TempDir()
	relativePath := "Data/config.ini"

	currentPath := filepath.Join(gameRoot, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(currentPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(currentPath, []byte("enabled=0\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current) error = %v", err)
	}

	desiredPath := filepath.Join(modRoot, "config.ini")
	if err := os.WriteFile(desiredPath, []byte("enabled=1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(desired) error = %v", err)
	}

	canonicalPath := deployment.CanonicalGameRelativePath(relativePath)
	entry := review.CachedPreview{
		Plan: planner.DeploymentPlan{
			Mode: planner.PlanModeFirstApply,
			Paths: map[string]planner.PathPlan{
				canonicalPath: {
					GameRelativePath: relativePath,
					Current: planner.FileStateSnapshot{
						Exists:    true,
						Label:     "Current on disk",
						SizeBytes: 10,
					},
					Desired: planner.FileStateSnapshot{
						Exists: true,
						Label:  "Desired profile",
					},
				},
			},
		},
		Desired: deployment.DesiredState{
			Files: map[string]deployment.DesiredFile{
				canonicalPath: {
					GameRelativePath: relativePath,
					SourcePath:       desiredPath,
				},
			},
		},
	}

	result, err := Inspect(entry, gameRoot, relativePath)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if result.Kind != InspectionTextDiff {
		t.Fatalf("Inspect() kind = %q, want text_diff", result.Kind)
	}
	if len(result.TextLines) == 0 {
		t.Fatal("Inspect() text lines = 0, want populated diff")
	}
}

func TestInspectImageWhenOnlyDesiredAvailable(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	modRoot := t.TempDir()
	relativePath := "backgrounds/xc1_night.jpg"

	desiredPath := filepath.Join(modRoot, "xc1_night.jpg")
	file, err := os.Create(desiredPath)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 8, 6))
	if err := jpeg.Encode(file, img, nil); err != nil {
		t.Fatalf("jpeg.Encode() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	canonicalPath := deployment.CanonicalGameRelativePath(relativePath)
	entry := review.CachedPreview{
		Plan: planner.DeploymentPlan{
			Mode: planner.PlanModeFirstApply,
			Paths: map[string]planner.PathPlan{
				canonicalPath: {
					GameRelativePath: relativePath,
					Current: planner.FileStateSnapshot{
						Exists: false,
						Label:  "Not present",
					},
					Desired: planner.FileStateSnapshot{
						Exists: true,
						Label:  "Desired content",
					},
				},
			},
		},
		Desired: deployment.DesiredState{
			Files: map[string]deployment.DesiredFile{
				canonicalPath: {
					GameRelativePath: relativePath,
					SourcePath:       desiredPath,
				},
			},
		},
	}

	result, err := Inspect(entry, gameRoot, relativePath)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if result.Kind != InspectionImageMetadata {
		t.Fatalf("Inspect() kind = %q, want image_metadata", result.Kind)
	}
	if result.ImageMetadataRight == nil || result.ImageMetadataRight.Format != "jpeg" || result.ImageMetadataRight.Width != 8 || result.ImageMetadataRight.Height != 6 {
		t.Fatalf("Inspect() right metadata = %+v, want jpeg 8x6 image", result.ImageMetadataRight)
	}
	if result.ImageMetadataLeft != nil {
		t.Fatalf("Inspect() left metadata = %+v, want nil for missing current file", result.ImageMetadataLeft)
	}
}
