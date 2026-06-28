package inspect

import (
	"fmt"
	"os"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/review"
)

func Inspect(
	entry review.CachedPreview,
	gameInstallPath string,
	relativePath string,
) (result InspectionResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("inspect deployment file: %w", err)
		}
	}()

	canonicalPath := deployment.CanonicalGameRelativePath(relativePath)
	pathPlan, found := entry.Plan.Paths[canonicalPath]
	if !found {
		return InspectionResult{}, fmt.Errorf("deployment path %q was not found in preview", relativePath)
	}

	resolvedPaths, err := resolveStatePaths(entry, gameInstallPath, relativePath)
	if err != nil {
		return InspectionResult{}, err
	}

	comparison := buildComparisonFromPathPlan(pathPlan)
	pair := SelectDefaultPair(string(entry.Plan.Mode), comparison)

	leftResolved := resolvedPaths[pair.Left]
	rightResolved := resolvedPaths[pair.Right]

	result = InspectionResult{
		RelativePath: pathPlan.GameRelativePath,
		LeftState:    pair.Left,
		RightState:   pair.Right,
		Left:         toSideMetadata(pair.Left, leftResolved),
		Right:        toSideMetadata(pair.Right, rightResolved),
	}

	fileClass := classifyRelativePath(pathPlan.GameRelativePath)
	result.Kind = inspectionKindForClass(fileClass)

	if !leftResolved.Available && !rightResolved.Available {
		result.Kind = InspectionBinaryFallback
		result.FallbackReason = unavailableCompareReason(leftResolved, rightResolved)
		return result, nil
	}

	if !leftResolved.Available || !rightResolved.Available {
		if result.Kind == InspectionTextDiff {
			result.Kind = InspectionBinaryFallback
			result.FallbackReason = unavailableCompareReason(leftResolved, rightResolved)
			return result, nil
		}
	}

	switch result.Kind {
	case InspectionTextDiff:
		return inspectText(result, leftResolved, rightResolved)
	case InspectionPEMetadata:
		return inspectPE(result, leftResolved, rightResolved)
	case InspectionImageMetadata:
		return inspectImage(result, leftResolved, rightResolved)
	case InspectionArchiveListing:
		return inspectArchive(result, leftResolved, rightResolved)
	default:
		return inspectBinary(result, leftResolved, rightResolved)
	}
}

func inspectText(result InspectionResult, left resolvedStatePath, right resolvedStatePath) (InspectionResult, error) {
	lines, limited, reason, err := buildTextDiff(left.Path, right.Path)
	if err != nil {
		if os.IsNotExist(err) {
			result.Kind = InspectionBinaryFallback
			result.FallbackReason = "Text diff is unavailable because one or both files could not be read."
			return result, nil
		}
		return InspectionResult{}, err
	}
	if limited {
		result.Kind = InspectionBinaryFallback
		result.LimitReached = true
		result.LimitReason = reason
		result.FallbackReason = reason
		return result, nil
	}

	result.TextLines = lines
	return result, nil
}

func inspectPE(result InspectionResult, left resolvedStatePath, right resolvedStatePath) (InspectionResult, error) {
	if left.Available {
		leftMetadata, leftErr := readPEMetadata(left.Path)
		if leftErr != nil {
			result.Kind = InspectionBinaryFallback
			result.FallbackReason = "PE metadata could not be read; showing hash and size only."
			return inspectBinary(result, left, right)
		}
		result.PEMetadataLeft = leftMetadata
	}

	if right.Available {
		rightMetadata, rightErr := readPEMetadata(right.Path)
		if rightErr != nil {
			result.Kind = InspectionBinaryFallback
			result.FallbackReason = "PE metadata could not be read; showing hash and size only."
			return inspectBinary(result, left, right)
		}
		result.PEMetadataRight = rightMetadata
	}

	if !left.Available || !right.Available {
		result.FallbackReason = unavailableCompareReason(left, right)
	}

	return result, nil
}

func inspectImage(result InspectionResult, left resolvedStatePath, right resolvedStatePath) (InspectionResult, error) {
	if left.Available {
		leftMetadata, leftReason, leftErr := imageMetadataOrFallback(left.Path, left)
		if leftErr != nil {
			return InspectionResult{}, fmt.Errorf("read image metadata: %w", leftErr)
		}
		result.ImageMetadataLeft = leftMetadata
		if leftReason != "" {
			result.FallbackReason = leftReason
		}
	}

	if right.Available {
		rightMetadata, rightReason, rightErr := imageMetadataOrFallback(right.Path, right)
		if rightErr != nil {
			return InspectionResult{}, fmt.Errorf("read image metadata: %w", rightErr)
		}
		result.ImageMetadataRight = rightMetadata
		if rightReason != "" {
			result.FallbackReason = firstNonEmpty(result.FallbackReason, rightReason)
		}
	}

	if !left.Available || !right.Available {
		result.FallbackReason = firstNonEmpty(result.FallbackReason, unavailableCompareReason(left, right))
	}

	return result, nil
}

func inspectArchive(result InspectionResult, left resolvedStatePath, right resolvedStatePath) (InspectionResult, error) {
	if left.Available {
		leftListing, err := listArchive(left.Path)
		if err != nil {
			result.Kind = InspectionBinaryFallback
			result.FallbackReason = "Archive listing is unavailable; showing hash and size only."
			return inspectBinary(result, left, right)
		}
		result.ArchiveEntriesLeft = leftListing.Entries
		result.LimitReached = leftListing.LimitReached
		result.LimitReason = leftListing.LimitReason
	}

	if right.Available {
		rightListing, err := listArchive(right.Path)
		if err != nil {
			result.Kind = InspectionBinaryFallback
			result.FallbackReason = "Archive listing is unavailable; showing hash and size only."
			return inspectBinary(result, left, right)
		}
		result.ArchiveEntriesRight = rightListing.Entries
		result.LimitReached = result.LimitReached || rightListing.LimitReached
		result.LimitReason = firstNonEmpty(result.LimitReason, rightListing.LimitReason)
	}

	if !left.Available || !right.Available {
		result.FallbackReason = unavailableCompareReason(left, right)
	}

	return result, nil
}

func inspectBinary(result InspectionResult, left resolvedStatePath, right resolvedStatePath) (InspectionResult, error) {
	leftWithHash, err := ensureHash(left.Path, left)
	if err != nil {
		return InspectionResult{}, err
	}
	rightWithHash, err := ensureHash(right.Path, right)
	if err != nil {
		return InspectionResult{}, err
	}

	result.Kind = InspectionBinaryFallback
	result.Left = binarySideMetadata(result.LeftState, leftWithHash)
	result.Right = binarySideMetadata(result.RightState, rightWithHash)
	if result.FallbackReason == "" {
		result.FallbackReason = "Showing hash and size only."
	}

	return result, nil
}

func unavailableCompareReason(left resolvedStatePath, right resolvedStatePath) string {
	if !left.Available && !right.Available {
		return "Neither side is available for inspection."
	}
	if !left.Available {
		return left.Reason
	}

	return right.Reason
}
