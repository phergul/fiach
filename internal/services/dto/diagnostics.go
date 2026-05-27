package dto

type ListDiagnosticLogsInput struct {
	Limit     int
	Operation string
	Level     string
}

type DiagnosticLogEntry struct {
	Timestamp string
	Level     string
	Operation string
	Message   string
	Details   map[string]string
}
