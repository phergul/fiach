import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type {
  OptiScalerCandidate,
  OptiScalerTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { OptiScalerTargetList } from './OptiScalerTargetList';

describe('OptiScalerTargetList', () => {
  it('separates managed targets and shows candidate validation evidence', () => {
    const onSelect = vi.fn();
    const candidate = {
      architecture: 'x64',
      evidence: ['common Unreal Win64 directory'],
      executableName: 'Game-Win64-Shipping.exe',
      executableRelativePath: 'Game/Binaries/Win64/Game-Win64-Shipping.exe',
      hasOptiScaler: true,
      hasReShade: true,
      managed: false,
      targetRelativePath: 'Game/Binaries/Win64',
    } as OptiScalerCandidate;
    const target = {
      ID: 1,
      ExecutableRelativePath: 'Other/Game.exe',
      GraphicsAPI: 'directx',
      ManagementOrigin: 'installed',
      ProxyFilename: 'dxgi.dll',
      ReleaseTag: 'v1',
      ReleaseVersion: 'v1',
      Status: 'managed',
      TargetRelativePath: 'Other',
    } as OptiScalerTarget;

    render(
      <OptiScalerTargetList
        candidates={[candidate]}
        disabled={false}
        onSelect={onSelect}
        release={{ digest: 'new-digest', tag: 'v2' } as never}
        selectedKey={null}
        targets={[target]}
      />,
    );

    expect(screen.getByText('x64')).toBeInTheDocument();
    expect(screen.getByText('common Unreal Win64 directory')).toBeInTheDocument();
    expect(screen.getByText('OptiScaler detected')).toBeInTheDocument();
    expect(screen.getByText('ReShade detected')).toBeInTheDocument();
    expect(screen.getByText('Update available')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /Game-Win64-Shipping.exe/ }));
    expect(onSelect).toHaveBeenCalledWith({
      candidate,
      target: null,
    });
  });
});
