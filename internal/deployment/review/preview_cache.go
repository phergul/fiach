package review

import (
	"sync"
	"time"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
)

type CachedPreview struct {
	PreviewHash string
	ProfileID   int64
	GameID      int64
	ProfileName string
	Plan        planner.DeploymentPlan
	Desired     deployment.DesiredState
	AppliedAt   *time.Time
	BuiltAt     time.Time
}

type PreviewCache struct {
	mu      sync.RWMutex
	entries map[string]CachedPreview
}

func NewPreviewCache() *PreviewCache {
	return &PreviewCache{
		entries: map[string]CachedPreview{},
	}
}

func (c *PreviewCache) Store(entry CachedPreview) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for hash, existing := range c.entries {
		if existing.ProfileID == entry.ProfileID {
			delete(c.entries, hash)
		}
	}

	c.entries[entry.PreviewHash] = entry
}

func (c *PreviewCache) Get(previewHash string) (CachedPreview, bool) {
	if c == nil {
		return CachedPreview{}, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.entries[previewHash]
	return entry, found
}
