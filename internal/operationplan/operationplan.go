package operationplan

type OperationType string

const (
	OperationTypeCopy            OperationType = "copy"
	OperationTypeReplace         OperationType = "replace"
	OperationTypeCreateDirectory OperationType = "create_directory"
)

type ModContext struct {
	ModID   int64
	ModName string
}

type Operation struct {
	Type       OperationType
	SourcePath *string
	TargetPath string
	BackupPath *string
	Conflict   bool
	Mod        ModContext
}

type OperationPlan struct {
	Operations []Operation
}
