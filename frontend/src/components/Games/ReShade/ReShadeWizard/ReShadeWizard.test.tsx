import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  Action,
  Architecture,
  BuildVariant,
  ManagementStatus,
  RenderingAPI,
  VariantProvenance,
} from '@bindings/github.com/phergul/fiach/internal/reshade/models';
import type { ManagedReShadeContentCatalogue } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  ApplyManagedReShadeAction,
  InspectManagedReShadePreset,
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

const contentCatalogue = (): ManagedReShadeContentCatalogue => ({
  addons: [
    {
      description: 'Depth buffer access add-on',
      downloadUrl: 'https://example.com/depth.addon',
      effectInstallPath: '',
      id: 'addon-depth',
      name: 'Depth add-on',
      repositoryUrl: 'https://example.com/addons/depth',
    },
  ],
  cached: true,
  effects: [
    {
      denyEffectFiles: [],
      description: 'Core utility shaders',
      downloadUrl: 'https://example.com/standard.zip',
      effectFiles: ['DisplayDepth.fx', 'Bloom.fx'],
      enabled: true,
      id: 'standard',
      installPath: 'reshade-shaders/Shaders',
      modifiable: true,
      name: 'Standard effects',
      repositoryUrl: 'https://example.com/standard',
      required: false,
      textureInstallPath: 'reshade-shaders/Textures',
    },
    {
      denyEffectFiles: [],
      description: 'Cinematic color shaders',
      downloadUrl: 'https://example.com/cinematic.zip',
      effectFiles: ['CinematicDOF.fx'],
      enabled: true,
      id: 'cinematic',
      installPath: 'reshade-shaders/Shaders',
      modifiable: true,
      name: 'Cinematic effects',
      repositoryUrl: 'https://example.com/cinematic',
      required: false,
      textureInstallPath: 'reshade-shaders/Textures',
    },
  ],
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

const configureSelection = (buildVariant: BuildVariant): ReShadeOperationSelection => ({
  action: Action.ActionConfigureContent,
  candidate: null,
  target: {
    ActiveRuntimeFilename: 'dxgi.dll',
    Architecture: Architecture.ArchitectureX64,
    BuildVariant: buildVariant,
    CreatedAt: '',
    ExecutableRelativePath: 'Bin/Game.exe',
    GameID: 1,
    ID: 1,
    LastVerifiedAt: null,
    ManagementOrigin: 'installed',
    Provenance: {},
    ProxyFilename: 'dxgi.dll',
    RenderingAPI: RenderingAPI.RenderingAPID3D11,
    RuntimeVersion: '6.7.3',
    Status: ManagementStatus.ManagementStatusManaged,
    TargetRelativePath: 'Bin',
    UpdatedAt: '',
    VariantProvenance: VariantProvenance.VariantProvenanceVerified,
  },
});

const mockPreview = () => {
  vi.mocked(PreviewManagedReShadeAction).mockResolvedValue({
    canApply: true,
    conflicts: [],
    desiredTarget: null,
    drift: [],
    operations: [],
    pathImpacts: [],
    previewHash: 'preview-hash',
    request: { action: Action.ActionConfigureContent },
    userContentDrift: [],
    warnings: [],
  } as never);
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

  it('selects full packages without explicit effect files and partial packages with effect files', async () => {
    mockPreview();

    render(
      <ReShadeWizard
        catalogue={contentCatalogue()}
        chainTargets={[]}
        gameID={1}
        onClose={vi.fn()}
        onRecoveryRequired={vi.fn()}
        onRefresh={vi.fn()}
        selection={configureSelection(BuildVariant.BuildVariantStandard)}
      />,
    );

    fireEvent.click(screen.getByRole('checkbox', { name: 'Select Standard effects' }));
    fireEvent.click(screen.getByRole('button', { name: 'Preview' }));

    await waitFor(() => expect(PreviewManagedReShadeAction).toHaveBeenCalledOnce());
    expect(vi.mocked(PreviewManagedReShadeAction).mock.calls[0][0].content).toEqual({
      effectPackages: [{ id: 'standard' }],
    });

    vi.mocked(PreviewManagedReShadeAction).mockClear();
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    fireEvent.click(screen.getByLabelText('Bloom.fx'));
    fireEvent.click(screen.getByRole('button', { name: 'Preview' }));

    await waitFor(() => expect(PreviewManagedReShadeAction).toHaveBeenCalledOnce());
    expect(vi.mocked(PreviewManagedReShadeAction).mock.calls[0][0].content).toEqual({
      effectPackages: [{ effectFiles: ['DisplayDepth.fx'], id: 'standard' }],
    });
  });

  it('clears all effects by removing the package selection', async () => {
    mockPreview();

    render(
      <ReShadeWizard
        catalogue={contentCatalogue()}
        chainTargets={[]}
        gameID={1}
        onClose={vi.fn()}
        onRecoveryRequired={vi.fn()}
        onRefresh={vi.fn()}
        selection={configureSelection(BuildVariant.BuildVariantStandard)}
      />,
    );

    fireEvent.click(screen.getByRole('checkbox', { name: 'Select Standard effects' }));
    fireEvent.click(screen.getByRole('button', { name: 'Clear all effects' }));
    fireEvent.click(screen.getByRole('button', { name: 'Preview' }));

    await waitFor(() => expect(PreviewManagedReShadeAction).toHaveBeenCalledOnce());
    expect(vi.mocked(PreviewManagedReShadeAction).mock.calls[0][0].content).toBeUndefined();
  });

  it('filters packages by effect file names', () => {
    render(
      <ReShadeWizard
        catalogue={contentCatalogue()}
        chainTargets={[]}
        gameID={1}
        onClose={vi.fn()}
        onRecoveryRequired={vi.fn()}
        onRefresh={vi.fn()}
        selection={configureSelection(BuildVariant.BuildVariantStandard)}
      />,
    );

    fireEvent.change(screen.getByRole('searchbox', { name: 'Search ReShade content' }), {
      target: { value: 'CinematicDOF.fx' },
    });

    expect(screen.getByRole('button', { name: /Cinematic effects/ })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Cinematic effects' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Standard effects/ })).not.toBeInTheDocument();
  });

  it('hides add-ons for standard builds and shows them for add-on builds', () => {
    const { rerender } = render(
      <ReShadeWizard
        catalogue={contentCatalogue()}
        chainTargets={[]}
        gameID={1}
        onClose={vi.fn()}
        onRecoveryRequired={vi.fn()}
        onRefresh={vi.fn()}
        selection={configureSelection(BuildVariant.BuildVariantStandard)}
      />,
    );

    expect(screen.queryByRole('tab', { name: 'Add-ons' })).not.toBeInTheDocument();

    rerender(
      <ReShadeWizard
        catalogue={contentCatalogue()}
        chainTargets={[]}
        gameID={1}
        onClose={vi.fn()}
        onRecoveryRequired={vi.fn()}
        onRefresh={vi.fn()}
        selection={configureSelection(BuildVariant.BuildVariantAddon)}
      />,
    );

    expect(screen.getByRole('tab', { name: 'Add-ons' })).toBeInTheDocument();
  });

  it('opens the preset helper and applies package recommendations', async () => {
    mockPreview();
    vi.mocked(InspectManagedReShadePreset).mockResolvedValue({
      missingEffects: [],
      recommendations: [
        {
          effectFiles: ['DisplayDepth.fx'],
          packageId: 'standard',
          packageName: 'Standard effects',
        },
      ],
      referencedEffects: ['DisplayDepth.fx'],
      warnings: [],
    });

    render(
      <ReShadeWizard
        catalogue={contentCatalogue()}
        chainTargets={[]}
        gameID={1}
        onClose={vi.fn()}
        onRecoveryRequired={vi.fn()}
        onRefresh={vi.fn()}
        selection={configureSelection(BuildVariant.BuildVariantStandard)}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Preset helper' }));
    fireEvent.change(screen.getByRole('textbox', { name: 'Preset path' }), {
      target: { value: 'ReShadePreset.ini' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Inspect' }));

    expect(await screen.findByText('1 referenced effects')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Add Standard effects' }));
    fireEvent.click(screen.getByRole('button', { name: 'Preview' }));

    await waitFor(() => expect(PreviewManagedReShadeAction).toHaveBeenCalledOnce());
    expect(vi.mocked(PreviewManagedReShadeAction).mock.calls[0][0].content).toEqual({
      effectPackages: [{ effectFiles: ['DisplayDepth.fx'], id: 'standard' }],
    });
  });
});
