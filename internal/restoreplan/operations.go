package restoreplan

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/phergul/fiach/internal/appliedstate"
)

func buildOperations(manifest appliedstate.ManifestDocument) []RestoreOperation {
	operations := make([]RestoreOperation, 0, len(manifest.AddedFiles)+len(manifest.ReplacedFiles)*2+len(manifest.CreatedDirectories))

	for _, entry := range manifest.AddedFiles {
		operations = append(operations, RestoreOperation{
			Type:                   RestoreOperationTypeRemoveAddedFile,
			ManifestOperationIndex: entry.OperationIndex,
			Mod:                    modFromManifest(entry.Mod),
			TargetPath:             entry.TargetPath,
		})
	}
	for _, entry := range manifest.ReplacedFiles {
		backupPath := entry.BackupPath
		operations = append(operations, RestoreOperation{
			Type:                   RestoreOperationTypeRestoreReplacedFile,
			ManifestOperationIndex: entry.OperationIndex,
			Mod:                    modFromManifest(entry.Mod),
			TargetPath:             entry.TargetPath,
			BackupPath:             &backupPath,
		})
	}

	directories := append([]appliedstate.CreatedDirectory(nil), manifest.CreatedDirectories...)
	sort.SliceStable(directories, func(i int, j int) bool {
		iPath := cleanPathForSort(directories[i].TargetPath)
		jPath := cleanPathForSort(directories[j].TargetPath)
		iDepth := pathDepth(iPath)
		jDepth := pathDepth(jPath)
		if iDepth != jDepth {
			return iDepth > jDepth
		}

		return iPath > jPath
	})
	for _, entry := range directories {
		operations = append(operations, RestoreOperation{
			Type:                   RestoreOperationTypeRemoveCreatedDir,
			ManifestOperationIndex: entry.OperationIndex,
			Mod:                    modFromManifest(entry.Mod),
			TargetPath:             entry.TargetPath,
		})
	}

	for _, entry := range manifest.ReplacedFiles {
		backupPath := entry.BackupPath
		operations = append(operations, RestoreOperation{
			Type:                   RestoreOperationTypeDeleteRestoredBackup,
			ManifestOperationIndex: entry.OperationIndex,
			Mod:                    modFromManifest(entry.Mod),
			TargetPath:             entry.TargetPath,
			BackupPath:             &backupPath,
		})
	}

	return operations
}

func modFromManifest(mod appliedstate.Mod) Mod {
	return Mod{
		ID:   mod.ID,
		Name: mod.Name,
	}
}

func cleanPathForSort(path string) string {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}

	return filepath.Clean(absolutePath)
}

func pathDepth(path string) int {
	volume := filepath.VolumeName(path)
	path = strings.TrimPrefix(path, volume)
	path = strings.Trim(path, string(filepath.Separator))
	if path == "" {
		return 0
	}

	return len(strings.Split(path, string(filepath.Separator)))
}
