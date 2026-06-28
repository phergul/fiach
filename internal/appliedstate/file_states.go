package appliedstate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/phergul/fiach/internal/fileops"
)

const OutputKindCopied = "copied"

const WinningSourceKindMod = "mod"

type PersistedFileState struct {
	GameID             int64
	GameRelativePath   string
	ProfileID          int64
	BaselineExists     bool
	BaselineSHA256     *string
	BaselineSizeBytes  *int64
	BaselineBackupPath *string
	AppliedExists      bool
	AppliedSHA256      *string
	AppliedSizeBytes   *int64
	WinningSourceKind  *string
	WinningSourceID    *string
	WinningModID       *int64
	WinningLoadOrder   *int64
	OutputKind         string
	UserDecision       *string
	LastAppliedAt      string
}

func AbsoluteToGameRelativePath(installPath string, absolutePath string) (string, error) {
	cleanInstallPath, err := fileops.CleanAbsPath("game install path", installPath)
	if err != nil {
		return "", err
	}
	cleanAbsolutePath, err := fileops.CleanAbsPath("managed file path", absolutePath)
	if err != nil {
		return "", err
	}
	if err := fileops.RequirePathWithinRoot("managed file path", cleanAbsolutePath, cleanInstallPath); err != nil {
		return "", err
	}

	relativePath, err := filepath.Rel(cleanInstallPath, cleanAbsolutePath)
	if err != nil {
		return "", fmt.Errorf("resolve game-relative path: %w", err)
	}

	return filepathToSlash(relativePath), nil
}

func filepathToSlash(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}
