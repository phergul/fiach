package appmode

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type Runtime struct {
	app *application.App
}

func NewRuntime(app *application.App) *Runtime {
	return &Runtime{app: app}
}

func (r *Runtime) OS() string {
	return runtime.GOOS
}
