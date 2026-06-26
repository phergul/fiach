package provenance

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/phergul/fiach/internal/deployment"
)

type pathConflict struct {
	category deployment.ConflictCategory
	message  string
}

func EnrichState(state *deployment.DesiredState, gameInstallPath string) error {
	if state == nil {
		return errors.New("desired state is not configured")
	}

	pathConflicts := detectPathSetConflicts(state.Files)
	gameInstallPath = filepath.Clean(gameInstallPath)

	canonicalPaths := sortedCanonicalPaths(state.Files)
	for _, canonicalPath := range canonicalPaths {
		file := state.Files[canonicalPath]
		file.Writers = FinalizeWriters(file.Writers)

		if conflict, found := pathConflicts[canonicalPath]; found {
			applyBlockedFile(&file, conflict.category, conflict.message)
			state.Files[canonicalPath] = file
			continue
		}

		winner, hasWinner := findModWinner(file.Writers)
		if !hasWinner {
			applyBlockedFile(&file, deployment.ConflictAmbiguousOverwrite, ExplainWinner(deployment.ConflictAmbiguousOverwrite, file.Writers, deployment.WriterEntry{}))
			state.Files[canonicalPath] = file
			continue
		}

		file.Winner = winner
		if len(modWriters(file.Writers)) > 1 {
			file.ConflictCategory = deployment.ConflictExpectedOverwrite
			file.RiskLevel = deployment.RiskInfo
		} else {
			file.ConflictCategory = ""
			file.RiskLevel = deployment.RiskNone
		}

		if err := applyGameInstallContext(&file, gameInstallPath); err != nil {
			return err
		}

		file.Explanation = ExplainWinner(file.ConflictCategory, file.Writers, winner)
		state.Files[canonicalPath] = file
	}

	return nil
}

func applyGameInstallContext(file *deployment.DesiredFile, gameInstallPath string) error {
	targetPath := filepath.Join(gameInstallPath, filepath.FromSlash(file.GameRelativePath))
	info, err := os.Stat(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			file.FileStatus = deployment.FileStatusAdded
			return nil
		}
		return fmt.Errorf("stat game install target %q: %w", targetPath, err)
	}

	if info.IsDir() {
		applyBlockedFile(file, deployment.ConflictDestructiveFileDirectory, fmt.Sprintf("Mod targets file path %q, but that path is an existing directory in the game install.", file.GameRelativePath))
		return nil
	}

	baseGame := NewBaseGameWriter()
	file.Writers = append([]deployment.WriterEntry{baseGame}, file.Writers...)
	file.Writers = RenumberWriterStack(file.Writers)
	file.FileStatus = deployment.FileStatusReplaced
	return nil
}

func applyBlockedFile(file *deployment.DesiredFile, category deployment.ConflictCategory, explanation string) {
	file.ConflictCategory = category
	file.FileStatus = deployment.FileStatusBlocked
	file.RiskLevel = deployment.RiskError
	file.Explanation = explanation
}

func findModWinner(writers []deployment.WriterEntry) (deployment.WriterEntry, bool) {
	for _, writer := range writers {
		if writer.SourceKind == deployment.SourceKindMod && writer.IsWinner {
			return writer, true
		}
	}
	return deployment.WriterEntry{}, false
}

func modWriters(writers []deployment.WriterEntry) []deployment.WriterEntry {
	result := make([]deployment.WriterEntry, 0, len(writers))
	for _, writer := range writers {
		if writer.SourceKind == deployment.SourceKindMod {
			result = append(result, writer)
		}
	}
	return result
}

func detectPathSetConflicts(files map[string]deployment.DesiredFile) map[string]pathConflict {
	conflicts := map[string]pathConflict{}
	paths := sortedCanonicalPaths(files)

	for leftIndex := 0; leftIndex < len(paths); leftIndex++ {
		for rightIndex := leftIndex + 1; rightIndex < len(paths); rightIndex++ {
			left := paths[leftIndex]
			right := paths[rightIndex]
			if deployment.IsStrictPathPrefix(left, right) {
				markPathConflict(conflicts, left, right, files[left].GameRelativePath, files[right].GameRelativePath)
			} else if deployment.IsStrictPathPrefix(right, left) {
				markPathConflict(conflicts, right, left, files[right].GameRelativePath, files[left].GameRelativePath)
			}
		}
	}

	return conflicts
}

func markPathConflict(conflicts map[string]pathConflict, filePath string, directoryPath string, fileLabel string, directoryLabel string) {
	message := fmt.Sprintf(
		"Path %q is targeted as a file while %q requires a parent directory at the same location.",
		fileLabel,
		directoryLabel,
	)
	conflicts[filePath] = pathConflict{
		category: deployment.ConflictDestructiveFileDirectory,
		message:  message,
	}
	conflicts[directoryPath] = pathConflict{
		category: deployment.ConflictDestructiveFileDirectory,
		message:  message,
	}
}

func sortedCanonicalPaths(files map[string]deployment.DesiredFile) []string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
