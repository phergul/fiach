package dto

type DevInfo struct {
	DataDir      string
	DatabasePath string
}

type DevLogEntry struct {
	Timestamp string
	Message   string
}
