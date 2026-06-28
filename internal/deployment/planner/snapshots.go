package planner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment/drift"
	"github.com/phergul/fiach/internal/fileops"
)

func baselineSnapshot(state appliedstate.PersistedFileState, hasApplied bool) FileStateSnapshot {
	if !hasApplied || !state.BaselineExists {
		return FileStateSnapshot{Exists: false}
	}

	snapshot := FileStateSnapshot{
		Exists: true,
		Label:  "Original content",
	}
	if state.BaselineSHA256 != nil {
		snapshot.SHA256 = *state.BaselineSHA256
	}
	if state.BaselineSizeBytes != nil {
		snapshot.SizeBytes = *state.BaselineSizeBytes
	}

	return snapshot
}

func appliedSnapshot(state appliedstate.PersistedFileState, hasApplied bool) FileStateSnapshot {
	if !hasApplied || !state.AppliedExists {
		return FileStateSnapshot{Exists: false}
	}

	snapshot := FileStateSnapshot{
		Exists: true,
		Label:  "Previously applied",
	}
	if state.AppliedSHA256 != nil {
		snapshot.SHA256 = *state.AppliedSHA256
	}
	if state.AppliedSizeBytes != nil {
		snapshot.SizeBytes = *state.AppliedSizeBytes
	}

	return snapshot
}

func baselineBackupPath(state appliedstate.PersistedFileState, hasApplied bool) string {
	if !hasApplied || state.BaselineBackupPath == nil {
		return ""
	}

	return *state.BaselineBackupPath
}

func currentSnapshotFromDrift(result drift.Result) FileStateSnapshot {
	if !result.CurrentExists {
		return FileStateSnapshot{
			Exists: false,
			Label:  "Current on disk",
		}
	}

	return FileStateSnapshot{
		Exists:    true,
		SHA256:    result.CurrentSHA256,
		SizeBytes: result.CurrentSizeBytes,
		Label:     "Current on disk",
	}
}

func readCurrentSnapshot(gameInstallPath string, gameRelativePath string) (FileStateSnapshot, error) {
	targetPath := filepath.Join(gameInstallPath, filepath.FromSlash(gameRelativePath))
	hash, size, err := fileops.FileIntegrity(targetPath)
	if errors.Is(err, os.ErrNotExist) {
		return FileStateSnapshot{
			Exists: false,
			Label:  "Current on disk",
		}, nil
	}
	if err != nil {
		return FileStateSnapshot{}, fmt.Errorf("hash current file %q: %w", gameRelativePath, err)
	}

	return FileStateSnapshot{
		Exists:    true,
		SHA256:    hash,
		SizeBytes: size,
		Label:     "Current on disk",
	}, nil
}

func SnapshotsMatch(left FileStateSnapshot, right FileStateSnapshot) bool {
	if left.Exists != right.Exists {
		return false
	}
	if !left.Exists {
		return true
	}

	return strings.EqualFold(left.SHA256, right.SHA256)
}

func SnapshotHash(snapshot FileStateSnapshot) string {
	if !snapshot.Exists {
		return ""
	}

	return snapshot.SHA256
}

func DiskMatchesApplied(appliedHash string, current FileStateSnapshot) bool {
	if appliedHash == "" {
		return false
	}
	if !current.Exists {
		return false
	}

	return strings.EqualFold(appliedHash, current.SHA256)
}
