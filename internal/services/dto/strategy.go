package dto

type StrategyType string
type StrategyVisibility string

const (
	StrategyTypeGenericCopy  StrategyType = "generic_copy"
	StrategyTypeReplaceFiles StrategyType = "replace_files"
	StrategyTypeBepInEx      StrategyType = "bepinex"
	StrategyTypeUnrealPak    StrategyType = "unreal_pak"
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

type Preview struct {
	StrategyType        StrategyType
	TargetBase          string
	TargetRelativePath  string
	TargetDisplayPath   string
	TotalFileCount      int
	TotalDirectoryCount int
	TargetFilePaths     []string
	IsCapped            bool
	Cap                 int
	Warnings            []string
}
