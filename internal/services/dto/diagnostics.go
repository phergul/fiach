package dto

type ListDiagnosticLogsInput struct {
	Limit     int
	Operation string
	Level     string
}

type ExportDiagnosticLogsInput struct {
	Path    string
	Entries []DiagnosticLogEntry
}

type DiagnosticOperation struct {
	Value string
	Label string
}

type DiagnosticOperationGroup struct {
	Area       string
	Operations []DiagnosticOperation
}

type DiagnosticLogEntry struct {
	Timestamp string
	Level     string
	Operation string
	Message   string
	Details   map[string]string
}
