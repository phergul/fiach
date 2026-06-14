//go:build !windows

package reshade

import (
	"context"
	"fmt"
)

type platformSetupProcessRunner struct{}

func (platformSetupProcessRunner) RunSetupProcess(
	context.Context,
	setupProcessRequest,
) (setupProcessResult, error) {
	return setupProcessResult{}, fmt.Errorf("run managed ReShade setup: unsupported platform")
}
