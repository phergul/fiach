package injection

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

type Store interface {
	ListOptiScalerTargets(context.Context, int64) ([]dbtypes.OptiScalerTarget, error)
	ListReShadeTargets(context.Context, int64) ([]dbtypes.ReShadeTarget, error)
}

type Coordinator struct {
	store Store
}

func NewCoordinator(store Store) *Coordinator {
	return &Coordinator{
		store: store,
	}
}

func (c *Coordinator) ListTargets(ctx context.Context, gameID int64) (targets []ChainTarget, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list injection targets: %w", err)
		}
	}()

	if c == nil || c.store == nil {
		return nil, errors.New("injection coordinator is not configured")
	}
	optiScalerTargets, err := c.store.ListOptiScalerTargets(ctx, gameID)
	if err != nil {
		return nil, err
	}
	reShadeTargets, err := c.store.ListReShadeTargets(ctx, gameID)
	if err != nil {
		return nil, err
	}

	byKey := map[string]*ChainTarget{}
	for _, target := range reShadeTargets {
		key := targetKey(target.TargetRelativePath)
		entry := byKey[key]
		if entry == nil {
			entry = &ChainTarget{
				GameID:                 target.GameID,
				TargetRelativePath:     target.TargetRelativePath,
				ExecutableRelativePath: target.ExecutableRelativePath,
				APIFamily:              APIFamilyDirectX,
				PrimaryOwner:           OwnerReShade,
				PrimaryProxyFilename:   target.ProxyFilename,
				Status:                 Status(target.Status),
			}
			byKey[key] = entry
		}
		entry.ReShade = &ReShadeState{
			PreferredProxyFilename: target.ProxyFilename,
			ActiveRuntimeFilename:  target.ProxyFilename,
			Target:                 target,
		}
	}
	for _, target := range optiScalerTargets {
		key := targetKey(target.TargetRelativePath)
		entry := byKey[key]
		if entry == nil {
			entry = &ChainTarget{
				GameID:                 target.GameID,
				TargetRelativePath:     target.TargetRelativePath,
				ExecutableRelativePath: target.ExecutableRelativePath,
				APIFamily:              APIFamily(target.GraphicsAPI),
				Status:                 Status(target.Status),
			}
			byKey[key] = entry
		}
		entry.APIFamily = APIFamily(target.GraphicsAPI)
		entry.PrimaryOwner = OwnerOptiScaler
		entry.PrimaryProxyFilename = target.ProxyFilename
		entry.OptiScaler = &OptiScalerState{
			ProxyFilename: target.ProxyFilename,
			Target:        target,
		}
		if entry.Status != StatusRecoveryRequired {
			entry.Status = Status(target.Status)
		}
	}

	targets = make([]ChainTarget, 0, len(byKey))
	for _, target := range byKey {
		targets = append(targets, *target)
	}
	return targets, nil
}

func (c *Coordinator) ListManagedOptiScalerTargets(ctx context.Context, gameID int64) ([]dbtypes.OptiScalerTarget, error) {
	if c == nil || c.store == nil {
		return nil, errors.New("injection coordinator is not configured")
	}
	return c.store.ListOptiScalerTargets(ctx, gameID)
}

func (c *Coordinator) ManagedDirectXOptiScalerTargets(ctx context.Context, gameID int64) (targets []dbtypes.OptiScalerTarget, blocked bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list managed DirectX OptiScaler injection targets: %w", err)
		}
	}()

	optiScalerTargets, err := c.ListManagedOptiScalerTargets(ctx, gameID)
	if err != nil {
		return nil, false, err
	}
	for _, target := range optiScalerTargets {
		switch APIFamily(target.GraphicsAPI) {
		case APIFamilyDirectX:
			targets = append(targets, target)
		case APIFamilyVulkan:
			blocked = true
		}
	}
	return targets, blocked, nil
}

func targetKey(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "."
	}
	return strings.ToLower(filepath.Clean(path))
}
