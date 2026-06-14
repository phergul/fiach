import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  ReShadeInstallerVariant,
  ReShadeSessionPhase,
} from '@bindings/github.com/phergul/fiach/internal/optiscaler/models';
import {
  ApplyOptiScalerReShadeRepair,
  CancelOptiScalerReShadeSession,
  GetOptiScalerReShadeSession,
  PreviewOptiScalerReShadeRepair,
  RescanOptiScalerReShadeSession,
  StartOptiScalerReShadeSession,
} from '@bindings/github.com/phergul/fiach/internal/services/optiscalerservice';

import { OptiScalerReShadeSession } from './OptiScalerReShadeSession';

vi.mock('@bindings/github.com/phergul/fiach/internal/services/optiscalerservice', () => ({
  ApplyOptiScalerReShadeRepair: vi.fn(),
  CancelOptiScalerReShadeSession: vi.fn(),
  GetOptiScalerReShadeSession: vi.fn(),
  PreviewOptiScalerReShadeRepair: vi.fn(),
  RescanOptiScalerReShadeSession: vi.fn(),
  StartOptiScalerReShadeSession: vi.fn(),
}));

vi.mock('@bindings/github.com/phergul/fiach/internal/services/reshadeservice', () => ({
  DetectGameReShade: vi.fn().mockResolvedValue({}),
}));

const target = {
  ExecutableRelativePath: 'Bin/Game.exe',
  GraphicsAPI: 'directx',
  ID: 4,
  ProxyFilename: 'dxgi.dll',
  TargetRelativePath: 'Bin',
} as never;

describe('OptiScalerReShadeSession', () => {
  beforeEach(() => {
    vi.mocked(GetOptiScalerReShadeSession).mockReset();
    vi.mocked(GetOptiScalerReShadeSession).mockResolvedValue(null);
    vi.mocked(StartOptiScalerReShadeSession).mockReset();
    vi.mocked(RescanOptiScalerReShadeSession).mockReset();
    vi.mocked(PreviewOptiScalerReShadeRepair).mockReset();
    vi.mocked(CancelOptiScalerReShadeSession).mockReset();
    vi.mocked(ApplyOptiScalerReShadeRepair).mockReset();
  });

  it('preselects a unique target and starts the chosen installer variant', async () => {
    vi.mocked(StartOptiScalerReShadeSession).mockResolvedValue({
      chainedFilename: 'ReShade64.dll',
      executableRelativePath: 'Bin/Game.exe',
      gameId: 1,
      id: 'session',
      installerVariant: ReShadeInstallerVariant.ReShadeInstallerVariantAddon,
      phase: ReShadeSessionPhase.ReShadeSessionPhaseAwaitingCompletion,
      proxyFilename: 'dxgi.dll',
      startedAt: '2026-06-14T00:00:00Z',
      targetRelativePath: 'Bin',
    } as never);

    render(
      <OptiScalerReShadeSession
        gameID={1}
        onActiveChange={vi.fn()}
        onRefresh={vi.fn().mockResolvedValue(undefined)}
        request={{
          targetRelativePath: null,
          variant: ReShadeInstallerVariant.ReShadeInstallerVariantAddon,
        }}
        targets={[target]}
      />,
    );

    const targetSelect = await screen.findByRole('combobox', { name: 'Managed DirectX target' });
    await waitFor(() => expect(targetSelect).toHaveValue('Bin'));
    fireEvent.click(screen.getByRole('button', { name: 'Open ReShade with Add-on Support installer' }));

    await waitFor(() => expect(StartOptiScalerReShadeSession).toHaveBeenCalledWith({
      gameId: 1,
      installerVariant: ReShadeInstallerVariant.ReShadeInstallerVariantAddon,
      targetRelativePath: 'Bin',
    }));
    expect(await screen.findByText(/Complete or close it before rescanning/)).toBeInTheDocument();
  });

  it('shows the exact conflict path after completion rescan', async () => {
    vi.mocked(GetOptiScalerReShadeSession).mockResolvedValue({
      chainedFilename: 'ReShade64.dll',
      executableRelativePath: 'Bin/Game.exe',
      gameId: 1,
      id: 'session',
      installerVariant: ReShadeInstallerVariant.ReShadeInstallerVariantStandard,
      phase: ReShadeSessionPhase.ReShadeSessionPhaseAwaitingCompletion,
      proxyFilename: 'dxgi.dll',
      startedAt: '2026-06-14T00:00:00Z',
      targetRelativePath: 'Bin',
    } as never);
    vi.mocked(RescanOptiScalerReShadeSession).mockResolvedValue({
      message: 'DLL ownership could not be determined safely.',
      outcome: 'conflict',
      session: {
        chainedFilename: 'ReShade64.dll',
        conflictingPath: 'C:\\Game\\Bin\\ReShade64.dll',
        executableRelativePath: 'Bin/Game.exe',
        gameId: 1,
        id: 'session',
        installerVariant: ReShadeInstallerVariant.ReShadeInstallerVariantStandard,
        phase: ReShadeSessionPhase.ReShadeSessionPhaseConflict,
        proxyFilename: 'dxgi.dll',
        startedAt: '2026-06-14T00:00:00Z',
        targetRelativePath: 'Bin',
      },
    } as never);

    render(
      <OptiScalerReShadeSession
        gameID={1}
        onActiveChange={vi.fn()}
        onRefresh={vi.fn().mockResolvedValue(undefined)}
        request={null}
        targets={[target]}
      />,
    );

    fireEvent.click(await screen.findByRole('button', { name: 'Installer finished, rescan' }));
    expect(await screen.findByText(/C:\\Game\\Bin\\ReShade64.dll/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Cancel session' })).toBeInTheDocument();
  });
});
