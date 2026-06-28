package dto

type ApplyDeploymentResult struct {
	Success        bool
	CompletedCount int
	SkippedCount   int
	Message        string
	RolledBack     bool
}

// ApplyIncrementalDeploymentResult is deprecated; use ApplyDeploymentResult.
type ApplyIncrementalDeploymentResult = ApplyDeploymentResult
