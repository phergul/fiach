package installconfig

import "testing"

func TestSelectableStrategiesReturnsOnlyGenericCopy(t *testing.T) {
	t.Parallel()

	strategies := SelectableStrategies()
	if len(strategies) != 1 {
		t.Fatalf("SelectableStrategies() length = %d, want 1: %+v", len(strategies), strategies)
	}

	strategy := strategies[0]
	if strategy.Type != StrategyTypeGenericCopy || strategy.Visibility != StrategyVisibilitySelectable || !strategy.RequiresTargetPath {
		t.Fatalf("SelectableStrategies()[0] = %+v, want selectable generic copy", strategy)
	}
}

func TestAllStrategiesIncludesFutureInternalDescriptors(t *testing.T) {
	t.Parallel()

	strategies := AllStrategies()
	byType := map[StrategyType]StrategyDescriptor{}
	for _, strategy := range strategies {
		byType[strategy.Type] = strategy
	}

	for _, strategyType := range []StrategyType{
		StrategyTypeGenericCopy,
		StrategyTypeReplaceFiles,
		StrategyTypeBepInEx,
		StrategyTypeUnrealPak,
	} {
		if _, found := byType[strategyType]; !found {
			t.Fatalf("AllStrategies() missing %q: %+v", strategyType, strategies)
		}
	}

	for _, strategyType := range []StrategyType{
		StrategyTypeReplaceFiles,
		StrategyTypeBepInEx,
		StrategyTypeUnrealPak,
	} {
		strategy := byType[strategyType]
		if strategy.Visibility != StrategyVisibilityInternal {
			t.Fatalf("future strategy %q = %+v, want internal descriptor", strategyType, strategy)
		}
	}
}

func TestValidateSelectableStrategy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		strategyType StrategyType
		wantError    bool
	}{
		{
			name:         "generic copy",
			strategyType: StrategyTypeGenericCopy,
			wantError:    false,
		},
		{
			name:         "missing",
			strategyType: "",
			wantError:    true,
		},
		{
			name:         "unknown",
			strategyType: StrategyType("unknown"),
			wantError:    true,
		},
		{
			name:         "disabled internal",
			strategyType: StrategyTypeBepInEx,
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateSelectableStrategy(tt.strategyType)
			if (err != nil) != tt.wantError {
				t.Fatalf("ValidateSelectableStrategy(%q) error = %v, wantError %v", tt.strategyType, err, tt.wantError)
			}
		})
	}
}
