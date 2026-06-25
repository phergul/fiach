import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { OptiScalerCandidate } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { Action } from '@bindings/github.com/phergul/fiach/internal/optiscaler/models';
import {
  ApplyOptiScalerAction,
  PreviewOptiScalerAction,
} from '@bindings/github.com/phergul/fiach/internal/services/optiscalerservice';

import { OptiScalerWizard } from './OptiScalerWizard';

vi.mock('@bindings/github.com/phergul/fiach/internal/services/optiscalerservice', () => ({
  ApplyOptiScalerAction: vi.fn(),
  GetOptiScalerRecoveryState: vi.fn(),
  PreviewOptiScalerAction: vi.fn(),
}));

describe('OptiScalerWizard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('uses visible defaults and requires target and warning acknowledgement', () => {
    const candidate = {
      architecture: 'x64',
      evidence: [],
      executableName: 'Game.exe',
      executableRelativePath: 'Bin/Game.exe',
      hasOptiScaler: false,
      hasReShade: false,
      managed: false,
      targetRelativePath: 'Bin',
    } as OptiScalerCandidate;

    render(
      <OptiScalerWizard
        gameID={1}
        onClose={vi.fn()}
        onRecoveryRequired={vi.fn()}
        onRefresh={vi.fn()}
        selection={{ action: Action.ActionInstall, candidate, target: null }}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.change(screen.getByRole('combobox', { name: 'Graphics API' }), {
      target: { value: 'directx' },
    });
    expect(screen.getByRole('combobox', { name: /Proxy filename/ })).toHaveValue('dxgi.dll');
    expect(screen.getByRole('textbox', { name: /Process filter/ })).toHaveValue('Game.exe');
    fireEvent.change(screen.getByRole('combobox', { name: 'DXGI spoofing' }), {
      target: { value: 'false' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));

    expect(screen.getByRole('button', { name: 'Preview' })).toBeDisabled();
    fireEvent.click(screen.getByLabelText(/I confirm that/));
    fireEvent.click(screen.getByLabelText(/I understand the online-game/));
    expect(screen.getByRole('button', { name: 'Preview' })).toBeEnabled();
  });

  it('rebuilds drift with backup-and-continue and applies the replacement preview hash', async () => {
    const candidate = {
      architecture: 'x64',
      evidence: [],
      executableName: 'Game.exe',
      executableRelativePath: 'Bin/Game.exe',
      hasOptiScaler: false,
      hasReShade: false,
      managed: false,
      targetRelativePath: 'Bin',
    } as OptiScalerCandidate;
    const previewMock = vi.mocked(PreviewOptiScalerAction);
    previewMock
      .mockResolvedValueOnce({
        canApply: false,
        configurationChanges: [],
        conflicts: ['Managed files have drifted'],
        drift: [{ expectedHash: 'a', missing: false, relativePath: 'dxgi.dll' }],
        operations: [],
        previewHash: 'blocked-hash',
        request: { action: Action.ActionInstall },
        warnings: [],
      } as never)
      .mockResolvedValueOnce({
        canApply: true,
        configurationChanges: [],
        conflicts: [],
        drift: [{ expectedHash: 'a', missing: false, relativePath: 'dxgi.dll' }],
        operations: [],
        previewHash: 'replacement-hash',
        request: { action: Action.ActionInstall },
        warnings: [],
      } as never);
    vi.mocked(ApplyOptiScalerAction).mockResolvedValue({
      message: 'Completed',
      rolledBack: false,
      success: true,
    });
    const onRefresh = vi.fn().mockResolvedValue(undefined);

    render(
      <OptiScalerWizard
        gameID={1}
        onClose={vi.fn()}
        onRecoveryRequired={vi.fn()}
        onRefresh={onRefresh}
        selection={{ action: Action.ActionInstall, candidate, target: null }}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.change(screen.getByRole('combobox', { name: 'Graphics API' }), {
      target: { value: 'directx' },
    });
    fireEvent.change(screen.getByRole('combobox', { name: 'DXGI spoofing' }), {
      target: { value: 'false' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByLabelText(/I confirm that/));
    fireEvent.click(screen.getByLabelText(/I understand the online-game/));
    fireEvent.click(screen.getByRole('button', { name: 'Preview' }));

    expect(await screen.findByText('Managed files have drifted')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Back up drift and rebuild preview' }));
    await waitFor(() => expect(previewMock).toHaveBeenCalledTimes(2));
    expect(previewMock.mock.calls[1][0].backupAndContinue).toBe(true);

    fireEvent.click(screen.getByRole('button', { name: 'Install' }));
    await waitFor(() => expect(ApplyOptiScalerAction).toHaveBeenCalled());
    expect(vi.mocked(ApplyOptiScalerAction).mock.calls[0][0].backupAndContinue).toBe(true);
    expect(vi.mocked(ApplyOptiScalerAction).mock.calls[0][1]).toBe('replacement-hash');
    expect(onRefresh).toHaveBeenCalledOnce();
  });

  it('uses adaptive steps for managed operations', () => {
    render(
      <OptiScalerWizard
        gameID={1}
        onClose={vi.fn()}
        onRecoveryRequired={vi.fn()}
        onRefresh={vi.fn()}
        selection={{
          action: Action.ActionRepair,
          candidate: null,
          target: {
            DXGISpoofing: false,
            EnableReShadeCoexistence: false,
            ExecutableRelativePath: 'Bin/Game.exe',
            GraphicsAPI: 'directx',
            ProcessFilter: 'Game.exe',
            ProxyFilename: 'dxgi.dll',
            TargetRelativePath: 'Bin',
          } as never,
        }}
      />,
    );

    expect(screen.queryByText('1. Target')).not.toBeInTheDocument();
    expect(screen.getByText('1. Configuration')).toBeInTheDocument();
    expect(screen.getByText('2. Preview')).toBeInTheDocument();
    expect(screen.getByText('3. Result')).toBeInTheDocument();
    expect(screen.queryByText('Safety')).not.toBeInTheDocument();
  });

  it('confirms before discarding changed values', () => {
    const onClose = vi.fn();
    const candidate = {
      architecture: 'x64',
      evidence: [],
      executableName: 'Game.exe',
      executableRelativePath: 'Bin/Game.exe',
      hasOptiScaler: false,
      hasReShade: false,
      managed: false,
      targetRelativePath: 'Bin',
    } as OptiScalerCandidate;

    render(
      <OptiScalerWizard
        gameID={1}
        onClose={onClose}
        onRecoveryRequired={vi.fn()}
        onRefresh={vi.fn()}
        selection={{ action: Action.ActionInstall, candidate, target: null }}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.change(screen.getByRole('combobox', { name: 'Graphics API' }), {
      target: { value: 'directx' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Close OptiScaler wizard' }));
    expect(screen.getByText('Discard OptiScaler changes?')).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Discard changes' }));
    expect(onClose).toHaveBeenCalledOnce();
  });
});
