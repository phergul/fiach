package updater

import (
	"github.com/phergul/fiach/internal/theme"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

const updaterWindowTitle = "Fiach"

func builtinWindow(themeID string) updater.WindowOption {
	return &updater.BuiltinWindow{
		CSS: theme.UpdaterCSS(themeID),
		Options: updater.WindowOptions{
			Title: updaterWindowTitle,
		},
	}
}
