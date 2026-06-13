import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type {
  OptiScalerCandidate,
  OptiScalerTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { Action } from '@bindings/github.com/phergul/fiach/internal/optiscaler/models';

import { OptiScalerDetail } from './OptiScalerDetail';

const renderDetail = (candidate: OptiScalerCandidate) => {
  const onStartAction = vi.fn();
  render(
    <OptiScalerDetail
      candidateCount={1}
      managedCount={0}
      onStartAction={onStartAction}
      release={null}
      selection={{ candidate, target: null }}
    />,
  );
  return onStartAction;
};

describe('OptiScalerDetail', () => {
  it('shows only install for a clean candidate', () => {
    const onStartAction = renderDetail({
      architecture: 'x64',
      evidence: [],
      executableName: 'Game.exe',
      executableRelativePath: 'Game.exe',
      hasOptiScaler: false,
      hasReShade: false,
      managed: false,
      targetRelativePath: '.',
    } as OptiScalerCandidate);

    expect(screen.queryByRole('button', { name: 'Adopt' })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Install' }));
    expect(onStartAction).toHaveBeenCalledWith(Action.ActionInstall);
  });

  it('shows only adopt when OptiScaler is detected', () => {
    const onStartAction = renderDetail({
      architecture: 'x64',
      evidence: [],
      executableName: 'Game.exe',
      executableRelativePath: 'Game.exe',
      hasOptiScaler: true,
      hasReShade: false,
      managed: false,
      targetRelativePath: '.',
    } as OptiScalerCandidate);

    expect(screen.queryByRole('button', { name: 'Install' })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Adopt' }));
    expect(onStartAction).toHaveBeenCalledWith(Action.ActionAdopt);
  });

  it('shows managed lifecycle actions', () => {
    const onStartAction = vi.fn();
    render(
      <OptiScalerDetail
        candidateCount={0}
        managedCount={1}
        onStartAction={onStartAction}
        release={null}
        selection={{
          candidate: null,
          target: {
            ExecutableRelativePath: 'Game.exe',
            GraphicsAPI: 'directx',
            ProxyFilename: 'dxgi.dll',
            ReleaseTag: 'v1',
            Status: 'managed',
            TargetRelativePath: '.',
          } as OptiScalerTarget,
        }}
      />,
    );

    expect(screen.getByRole('button', { name: 'Update' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Repair' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Uninstall' })).toBeInTheDocument();
  });
});
