package appmode

import (
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func DataRoot() string {
	return filepath.Join(application.Path(application.PathDataHome), DataDirName())
}

func CacheRoot() string {
	return filepath.Join(application.Path(application.PathCacheHome), DataDirName())
}
