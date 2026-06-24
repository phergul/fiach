import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type {
  ReShadeDiscoveryResult,
  ReShadeTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { ReShadeTargetTable } from './ReShadeTargetTable';

const candidate = (
  overrides: Partial<ReShadeDiscoveryResult['candidates'][number]> = {},
) => ({
  apiOptions: [
    { proxies: ['d3d9.dll'], renderingApi: 'd3d9' },
    { proxies: ['dxgi.dll'], renderingApi: 'd3d10' },
    { proxies: ['dxgi.dll'], renderingApi: 'd3d11' },
    { proxies: ['dxgi.dll'], renderingApi: 'd3d12' },
  ],
  architecture: 'x64',
  conflicts: [],
  executableRelativePath: 'Bin/Game.exe',
  proxyEvidence: [],
  targetRelativePath: 'Bin',
  ...overrides,
} as ReShadeDiscoveryResult['candidates'][number]);

const target = (overrides: Partial<ReShadeTarget> = {}) => ({
  ActiveRuntimeFilename: 'ReShade64.dll',
  Architecture: 'x64',
  BuildVariant: 'standard',
  ExecutableRelativePath: 'Bin/Game.exe',
  ID: 1,
  ProxyFilename: 'dxgi.dll',
  RenderingAPI: 'd3d11',
  RuntimeVersion: '6.7.3',
  Status: 'managed',
  TargetRelativePath: 'Bin',
  VariantProvenance: 'verified',
  ...overrides,
} as ReShadeTarget);

describe('ReShadeTargetTable', () => {
  it('keeps detected API selection out of the row and suppresses clean placeholders', () => {
    render(
      <ReShadeTargetTable
        chainTargets={[]}
        disabled={false}
        discovery={{ candidates: [candidate()], warnings: [] } as ReShadeDiscoveryResult}
        installerStatus={null}
        onStartOperation={vi.fn()}
        targets={[]}
      />,
    );

    expect(screen.getByText('DirectX target')).toBeInTheDocument();
    expect(screen.queryByText('D3D9, D3D10, D3D11, D3D12')).not.toBeInTheDocument();
    expect(screen.queryByText('No runtime found')).not.toBeInTheDocument();
    expect(screen.queryByText('No managed chain')).not.toBeInTheDocument();
    expect(screen.queryByText('Ready')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Install' })).toBeInTheDocument();
  });

  it('shows managed runtime once while keeping the exact rendering API visible', () => {
    render(
      <ReShadeTargetTable
        chainTargets={[]}
        disabled={false}
        discovery={null}
        installerStatus={null}
        onStartOperation={vi.fn()}
        targets={[target()]}
      />,
    );

    expect(screen.getByText('D3D11')).toBeInTheDocument();
    expect(screen.getAllByText('v6.7.3')).toHaveLength(1);
    expect(screen.getByRole('button', { name: 'Content' })).toBeInTheDocument();
  });
});
