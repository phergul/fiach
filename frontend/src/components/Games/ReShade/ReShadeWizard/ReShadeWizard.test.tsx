import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  Action,
  Architecture,
  RenderingAPI,
} from '@bindings/github.com/phergul/fiach/internal/reshade/models';
import type { ManagedReShadeContentCatalogue } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  ApplyManagedReShadeAction,
  PreviewManagedReShadeAction,
} from '@bindings/github.com/phergul/fiach/internal/services/reshadeservice';

import { ReShadeWizard } from './ReShadeWizard';
import type { ReShadeOperationSelection } from '../ReShadeTargetTable/ReShadeTargetTable';

vi.mock('@bindings/github.com/phergul/fiach/internal/services/reshadeservice', () => ({
  ApplyManagedReShadeAction: vi.fn(),
  GetManagedReShadeRecoveryState: vi.fn(),
  InspectManagedReShadePreset: vi.fn(),
  ListManagedReShadeContentCatalogue: vi.fn(),
  PreviewManagedReShadeAction: vi.fn(),
}));

const emptyCatalogue = (cached: boolean): ManagedReShadeContentCatalogue => ({
  addons: [],
  cached,
  effects: [],
});

const installSelection: ReShadeOperationSelection = {
  action: Action.ActionInstall,
  candidate: {
    apiOptions: [
      {
        proxies: ['dxgi.dll'],
        renderingApi: RenderingAPI.RenderingAPID3D11,
      },
    ],
    architecture: Architecture.ArchitectureX64,
    conflicts: [],
    executableRelativePath: 'Bin/Game.exe',
    proxyEvidence: [],
    targetRelativePath: 'Bin',
  },
  target: null,
};

describe('ReShadeWizard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('keeps the success result visible when refreshed catalogue props arrive after apply', async () => {
    vi.mocked(PreviewManagedReShadeAction).mockResolvedValue({
      canApply: true,
      conflicts: [],
      desiredTarget: {
        managementOrigin: 'installed',
        manifest: {
          files: [],
          hasPreAdoptionRollbackData: false,
          version: 1,
        },
        provenance: {},
        runtimeVersion: '6.7.3',
      },
      drift: [],
      operations: [],
      pathImpacts: [],
      previewHash: 'preview-hash',
      request: { action: Action.ActionInstall },
      userContentDrift: [],
      warnings: [],
    } as never);
    vi.mocked(ApplyManagedReShadeAction).mockResolvedValue({
      message: 'Completed',
      rolledBack: false,
      success: true,
    });
    const onRefresh = vi.fn().mockResolvedValue(undefined);
    const onClose = vi.fn();

    const { rerender } = render(
      <ReShadeWizard
        catalogue={emptyCatalogue(false)}
        chainTargets={[]}
        gameID={1}
        onClose={onClose}
        onRecoveryRequired={vi.fn()}
        onRefresh={onRefresh}
        selection={installSelection}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Preview' }));

    expect(await screen.findByRole('button', { name: 'Install' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Install' }));

    await waitFor(() => expect(ApplyManagedReShadeAction).toHaveBeenCalledOnce());
    expect(await screen.findByText('Operation complete')).toBeInTheDocument();
    expect(onRefresh).toHaveBeenCalledOnce();

    rerender(
      <ReShadeWizard
        catalogue={emptyCatalogue(true)}
        chainTargets={[]}
        gameID={1}
        onClose={onClose}
        onRecoveryRequired={vi.fn()}
        onRefresh={onRefresh}
        selection={installSelection}
      />,
    );

    expect(screen.getByText('Operation complete')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Done' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Next' })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Done' }));
    expect(onClose).toHaveBeenCalledOnce();
  });
});
