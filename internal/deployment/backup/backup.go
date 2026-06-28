package backup

import (
	"path/filepath"
)

const RootDirName = "deployment-backups"

func PathForTarget(gameModStoragePath string, gameRelativeTargetPath string) string {
	return filepath.Join(gameModStoragePath, RootDirName, filepath.FromSlash(gameRelativeTargetPath))
}
