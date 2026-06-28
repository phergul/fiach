package dto

type ApplyIncrementalDeploymentResult struct {
	Success        bool
	CompletedCount int
	SkippedCount   int
	Message        string
	RolledBack     bool
}
