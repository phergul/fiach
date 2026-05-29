package installconfig

import (
	"errors"
	"fmt"
)

type StrategyType string
type StrategyVisibility string

const (
	StrategyTypeGenericCopy StrategyType = "generic_copy"
	StrategyTypeBepInEx     StrategyType = "bepinex"
	StrategyTypeUnrealPak   StrategyType = "unreal_pak"
)

const (
	StrategyVisibilitySelectable StrategyVisibility = "selectable"
	StrategyVisibilityDisabled   StrategyVisibility = "disabled"
	StrategyVisibilityInternal   StrategyVisibility = "internal"
)

type StrategyDescriptor struct {
	Type               StrategyType
	Label              string
	Description        string
	Visibility         StrategyVisibility
	RequiresTargetPath bool
}

var strategies = []StrategyDescriptor{
	{
		Type:               StrategyTypeGenericCopy,
		Label:              "File Copy",
		Description:        "Install mod files into the game folder. Existing files are backed up before being replaced.",
		Visibility:         StrategyVisibilitySelectable,
		RequiresTargetPath: true,
	},
	{
		Type:               StrategyTypeBepInEx,
		Label:              "BepInEx",
		Description:        "Install plugin files into a BepInEx layout.",
		Visibility:         StrategyVisibilityInternal,
		RequiresTargetPath: true,
	},
	{
		Type:               StrategyTypeUnrealPak,
		Label:              "Unreal PAK",
		Description:        "Install pak files into an Unreal Engine pak folder.",
		Visibility:         StrategyVisibilityInternal,
		RequiresTargetPath: true,
	},
}

func AllStrategies() []StrategyDescriptor {
	return cloneStrategies(strategies)
}

func SelectableStrategies() []StrategyDescriptor {
	selectable := make([]StrategyDescriptor, 0, len(strategies))
	for _, strategy := range strategies {
		if strategy.Visibility == StrategyVisibilitySelectable {
			selectable = append(selectable, strategy)
		}
	}

	return selectable
}

func ValidateSelectableStrategy(strategyType StrategyType) error {
	if strategyType == "" {
		return errors.New("import strategy is required")
	}

	for _, strategy := range strategies {
		if strategy.Type != strategyType {
			continue
		}
		if strategy.Visibility != StrategyVisibilitySelectable {
			return fmt.Errorf("import strategy %q is not supported", strategyType)
		}

		return nil
	}

	return fmt.Errorf("unknown import strategy %q", strategyType)
}

func cloneStrategies(source []StrategyDescriptor) []StrategyDescriptor {
	clone := make([]StrategyDescriptor, len(source))
	copy(clone, source)
	return clone
}
