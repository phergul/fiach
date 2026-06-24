package optiscaler

import (
	"fmt"
	"strings"

	"github.com/phergul/fiach/internal/iniedit"
)

func UpdateManagedINI(contents []byte, config ManagedConfig) (updated []byte, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("update OptiScaler INI settings: %w", err)
		}
	}()

	document, err := iniedit.ParsePreserving(contents)
	if err != nil {
		return nil, err
	}
	document.SetSingleKey("Plugins", "LoadReshade", boolINI(config.LoadReShade))
	document.SetSingleKey("Spoofing", "Dxgi", boolINI(config.DXGISpoofing))
	document.SetSingleKey("ProcessFilter", "TargetProcessName", optionalINI(config.TargetProcessName))
	document.SetSingleKey("Hotfix", "CheckForUpdate", "false")
	return document.Bytes(), nil
}

func boolINI(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func optionalINI(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
