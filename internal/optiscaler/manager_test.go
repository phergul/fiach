package optiscaler

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

type memoryStore struct {
	mu      sync.Mutex
	nextID  int64
	targets map[string]dbtypes.OptiScalerTarget
}

func newMemoryStore() *memoryStore {
	return &memoryStore{nextID: 1, targets: map[string]dbtypes.OptiScalerTarget{}}
}

func (s *memoryStore) key(gameID int64, path string) string {
	return fmt.Sprintf("%d:%s", gameID, strings.ToLower(filepath.Clean(path)))
}

func (s *memoryStore) GetOptiScalerTarget(_ context.Context, gameID int64, path string) (dbtypes.OptiScalerTarget, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	target, found := s.targets[s.key(gameID, path)]
	return target, found, nil
}

func (s *memoryStore) ListOptiScalerTargets(_ context.Context, gameID int64) ([]dbtypes.OptiScalerTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var targets []dbtypes.OptiScalerTarget
	for _, target := range s.targets {
		if target.GameID == gameID {
			targets = append(targets, target)
		}
	}
	return targets, nil
}

func (s *memoryStore) SaveOptiScalerTarget(_ context.Context, input dbtypes.SaveOptiScalerTargetInput) (dbtypes.OptiScalerTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := s.key(input.GameID, input.TargetRelativePath)
	target := s.targets[key]
	if target.ID == 0 {
		target.ID = s.nextID
		s.nextID++
	}
	target.GameID = input.GameID
	target.TargetRelativePath = input.TargetRelativePath
	target.ExecutableRelativePath = input.ExecutableRelativePath
	target.GraphicsAPI = input.GraphicsAPI
	target.ProxyFilename = input.ProxyFilename
	target.DXGISpoofing = input.DXGISpoofing
	target.ProcessFilter = input.ProcessFilter
	target.ReleaseTag = input.ReleaseTag
	target.ReleaseVersion = input.ReleaseVersion
	target.ReleaseAssetName = input.ReleaseAssetName
	target.ReleaseDigest = input.ReleaseDigest
	target.ManagementOrigin = input.ManagementOrigin
	target.Status = input.Status
	target.ManifestJSON = input.ManifestJSON
	target.WarningVersion = input.WarningVersion
	target.WarningAcknowledgedAt = input.WarningAcknowledgedAt
	target.LastVerifiedAt = input.LastVerifiedAt
	s.targets[key] = target
	return target, nil
}

func (s *memoryStore) DeleteOptiScalerTarget(_ context.Context, gameID int64, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.targets, s.key(gameID, path))
	return nil
}
