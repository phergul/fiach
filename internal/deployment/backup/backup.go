package backup

import (
	"path/filepath"
)

const RootDirName = "operation-backups"

func PathForTarget(gameModStoragePath string, gameRelativeTargetPath string) string {
	return filepath.Join(gameModStoragePath, RootDirName, filepath.FromSlash(gameRelativeTargetPath))
}
