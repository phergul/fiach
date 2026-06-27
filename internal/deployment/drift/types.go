package drift

import "github.com/phergul/fiach/internal/deployment"

type Result struct {
	GameRelativePath string
	Kind             deployment.DriftKind
	CurrentExists    bool
	CurrentSHA256    string
	CurrentSizeBytes int64
	AppliedExists    bool
	AppliedSHA256    string
	AppliedSizeBytes int64
}
