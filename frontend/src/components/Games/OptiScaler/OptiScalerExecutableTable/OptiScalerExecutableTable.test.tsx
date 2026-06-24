import { fireEvent, render, screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { Action } from '@bindings/github.com/phergul/fiach/internal/optiscaler/models';
import type {
  OptiScalerCandidate,
  OptiScalerTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { OptiScalerExecutableTable } from './OptiScalerExecutableTable';

const candidate = (overrides: Partial<OptiScalerCandidate> = {}) => ({
  architecture: 'x64',
  evidence: [],
  executableName: 'Game.exe',
  executableRelativePath: 'Bin/Game.exe',
  hasOptiScaler: false,
  hasReShade: false,
  managed: false,
  targetRelativePath: 'Bin',
  ...overrides,
} as OptiScalerCandidate);

const target = (overrides: Partial<OptiScalerTarget> = {}) => ({
  ID: 1,
  ExecutableRelativePath: 'Managed/Game.exe',
  GraphicsAPI: 'directx',
  ProxyFilename: 'dxgi.dll',
  ReleaseDigest: 'current',
  ReleaseTag: 'v1',
  Status: 'managed',
  TargetRelativePath: 'Managed',
  ...overrides,
} as OptiScalerTarget);

describe('OptiScalerExecutableTable', () => {
  it('orders managed rows by attention and selects state-driven actions', () => {
    const onStartOperation = vi.fn();
    render(
      <OptiScalerExecutableTable
        candidates={[]}
        disabled={false}
        onStartOperation={onStartOperation}
        release={{ digest: 'new', tag: 'v2' } as never}
        targets={[
          target({ ID: 1, ExecutableRelativePath: 'Healthy.exe', ReleaseDigest: 'new', ReleaseTag: 'v2' }),
          target({ ID: 2, ExecutableRelativePath: 'Outdated.exe' }),
          target({ ID: 3, ExecutableRelativePath: 'Drifted.exe', Status: 'drifted' }),
        ]}
      />,
    );

    const rows = screen.getAllByText(/\.exe$/).map((element) => element.textContent);
    expect(rows.slice(0, 3)).toEqual(['Drifted.exe', 'Outdated.exe', 'Healthy.exe']);
    fireEvent.click(screen.getByRole('button', { name: 'Repair' }));
    expect(onStartOperation).toHaveBeenCalledWith(expect.objectContaining({ action: Action.ActionRepair }));
    fireEvent.click(screen.getByRole('button', { name: 'Update' }));
    expect(onStartOperation).toHaveBeenCalledWith(expect.objectContaining({ action: Action.ActionUpdate }));
  });

  it('uses install and adopt for unmanaged executables and disables actions during recovery', () => {
    const onStartOperation = vi.fn();
    const { rerender } = render(
      <OptiScalerExecutableTable
        candidates={[
          candidate(),
          candidate({ executableName: 'Existing.exe', executableRelativePath: 'Bin/Existing.exe', hasOptiScaler: true }),
        ]}
        disabled={false}
        onStartOperation={onStartOperation}
        release={null}
        targets={[]}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Install' }));
    fireEvent.click(screen.getByRole('button', { name: 'Adopt' }));
    expect(onStartOperation.mock.calls.map(([selection]) => selection.action)).toEqual([
      Action.ActionInstall,
      Action.ActionAdopt,
    ]);

    rerender(
      <OptiScalerExecutableTable
        candidates={[candidate()]}
        disabled
        onStartOperation={onStartOperation}
        release={null}
        targets={[]}
      />,
    );
    expect(within(screen.getByRole('button', { name: 'Install' })).getByText('Install')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Install' })).toBeDisabled();
  });

  it('suppresses detected row placeholders and generic validation evidence', () => {
    render(
      <OptiScalerExecutableTable
        candidates={[
          candidate({
            evidence: ['validated Windows x64 executable'],
          }),
        ]}
        disabled={false}
        onStartOperation={vi.fn()}
        release={null}
        targets={[]}
      />,
    );

    expect(screen.queryByText('Select during install')).not.toBeInTheDocument();
    expect(screen.queryByText('validated Windows x64 executable')).not.toBeInTheDocument();
    expect(screen.queryByText('Ready')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Install' })).toBeInTheDocument();
  });
});
