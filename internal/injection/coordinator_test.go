package injection

import (
	"context"
	"testing"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

type memoryStore struct {
	optiScaler []dbtypes.OptiScalerTarget
	reShade    []dbtypes.ReShadeTarget
}

func (s memoryStore) ListOptiScalerTargets(context.Context, int64) ([]dbtypes.OptiScalerTarget, error) {
	return s.optiScaler, nil
}

func (s memoryStore) ListReShadeTargets(context.Context, int64) ([]dbtypes.ReShadeTarget, error) {
	return s.reShade, nil
}

func TestCoordinatorCombinesProductsByTargetAndKeepsOptiScalerPrimary(t *testing.T) {
	t.Parallel()

	coordinator := NewCoordinator(memoryStore{
		optiScaler: []dbtypes.OptiScalerTarget{{
			GameID:                 1,
			TargetRelativePath:     ".",
			ExecutableRelativePath: "Game.exe",
			GraphicsAPI:            "directx",
			ProxyFilename:          "dxgi.dll",
			Status:                 "managed",
		}},
		reShade: []dbtypes.ReShadeTarget{{
			GameID:                 1,
			TargetRelativePath:     ".",
			ExecutableRelativePath: "Game.exe",
			RenderingAPI:           "d3d11",
			ProxyFilename:          "ReShade64.dll",
			Status:                 "managed",
		}},
	})

	targets, err := coordinator.ListTargets(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListTargets() error = %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("ListTargets() length = %d, want 1", len(targets))
	}
	target := targets[0]
	if target.PrimaryOwner != OwnerOptiScaler || target.PrimaryProxyFilename != "dxgi.dll" {
		t.Fatalf("primary = %q %q, want OptiScaler dxgi.dll", target.PrimaryOwner, target.PrimaryProxyFilename)
	}
	if target.OptiScaler == nil || target.ReShade == nil {
		t.Fatalf("combined target missing product state: %+v", target)
	}
}

func TestCoordinatorReportsDirectXTargetsAndVulkanBlock(t *testing.T) {
	t.Parallel()

	coordinator := NewCoordinator(memoryStore{
		optiScaler: []dbtypes.OptiScalerTarget{
			{GameID: 1, TargetRelativePath: "bin", GraphicsAPI: "directx"},
			{GameID: 1, TargetRelativePath: "vk", GraphicsAPI: "vulkan"},
		},
	})

	targets, blocked, err := coordinator.ManagedDirectXOptiScalerTargets(context.Background(), 1)
	if err != nil {
		t.Fatalf("ManagedDirectXOptiScalerTargets() error = %v", err)
	}
	if !blocked {
		t.Fatal("blocked = false, want true for Vulkan target")
	}
	if len(targets) != 1 || targets[0].TargetRelativePath != "bin" {
		t.Fatalf("targets = %+v, want only DirectX target", targets)
	}
}

func TestCoordinatorReportsOpenGLReShadeAPIFamily(t *testing.T) {
	t.Parallel()

	coordinator := NewCoordinator(memoryStore{
		reShade: []dbtypes.ReShadeTarget{{
			GameID:                 1,
			TargetRelativePath:     ".",
			ExecutableRelativePath: "Game.exe",
			RenderingAPI:           "opengl",
			ProxyFilename:          "opengl32.dll",
			Status:                 "managed",
		}},
	})

	targets, err := coordinator.ListTargets(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListTargets() error = %v", err)
	}
	if len(targets) != 1 ||
		targets[0].APIFamily != APIFamilyOpenGL ||
		targets[0].PrimaryProxyFilename != "opengl32.dll" {
		t.Fatalf("targets = %+v", targets)
	}
}
