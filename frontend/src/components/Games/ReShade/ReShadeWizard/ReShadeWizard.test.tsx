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
import type { ReShadeContentCatalogue } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  ApplyReShadeAction,
  InspectReShadePreset,
  ListReShadeContentCatalogue,
  PreviewReShadeAction,
} from '@bindings/github.com/phergul/fiach/internal/services/reshadeservice';
import { openReShadePreset } from '@utils';

import { ReShadeWizard } from './ReShadeWizard';
import type { ReShadeOperationSelection } from '../ReShadeTargetTable/ReShadeTargetTable';

vi.mock('@bindings/github.com/phergul/fiach/internal/services/reshadeservice', () => ({
  ApplyReShadeAction: vi.fn(),
  GetReShadeRecoveryState: vi.fn(),
  InspectReShadePreset: vi.fn(),
  ListReShadeContentCatalogue: vi.fn(),
  PreviewReShadeAction: vi.fn(),
}));

vi.mock('@utils', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@utils')>();

  return {
    ...actual,
    openReShadePreset: vi.fn(),
  };
});

const emptyCatalogue = (cached: boolean): ReShadeContentCatalogue => ({
  addons: [],
  cached,
  effects: [],
});

const contentCatalogue = (): ReShadeContentCatalogue => ({
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
  vi.mocked(PreviewReShadeAction).mockResolvedValue({
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
    vi.mocked(openReShadePreset).mockResolvedValue(null);
  });

  it('keeps the success result visible when refreshed catalogue props arrive after apply', async () => {
    vi.mocked(PreviewReShadeAction).mockResolvedValue({
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
    vi.mocked(ApplyReShadeAction).mockResolvedValue({
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

    await waitFor(() => expect(ApplyReShadeAction).toHaveBeenCalledOnce());
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

    await waitFor(() => expect(PreviewReShadeAction).toHaveBeenCalledOnce());
    expect(vi.mocked(PreviewReShadeAction).mock.calls[0][0].content).toEqual({
      effectPackages: [{ id: 'standard' }],
    });

    vi.mocked(PreviewReShadeAction).mockClear();
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    fireEvent.click(screen.getByLabelText('Bloom.fx'));
    fireEvent.click(screen.getByRole('button', { name: 'Preview' }));

    await waitFor(() => expect(PreviewReShadeAction).toHaveBeenCalledOnce());
    expect(vi.mocked(PreviewReShadeAction).mock.calls[0][0].content).toEqual({
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

    await waitFor(() => expect(PreviewReShadeAction).toHaveBeenCalledOnce());
    expect(vi.mocked(PreviewReShadeAction).mock.calls[0][0].content).toBeUndefined();
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
    vi.mocked(openReShadePreset).mockResolvedValue('ReShadePreset.ini');
    vi.mocked(InspectReShadePreset).mockResolvedValue({
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
    fireEvent.click(screen.getByRole('button', { name: 'Select preset' }));

    expect(await screen.findByText('1 referenced effects')).toBeInTheDocument();
    expect(InspectReShadePreset).toHaveBeenCalledWith(1, 'Bin', 'ReShadePreset.ini');
    fireEvent.click(screen.getByRole('button', { name: 'Add Standard effects' }));
    fireEvent.click(screen.getByRole('button', { name: 'Preview' }));

    await waitFor(() => expect(PreviewReShadeAction).toHaveBeenCalledOnce());
    expect(vi.mocked(PreviewReShadeAction).mock.calls[0][0].content).toEqual({
      effectPackages: [{ effectFiles: ['DisplayDepth.fx'], id: 'standard' }],
    });
  });

  it('keeps the content step visible after selecting a scrolled package', () => {
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

    fireEvent.click(screen.getByRole('checkbox', { name: 'Select Cinematic effects' }));

    expect(screen.getByRole('searchbox', { name: 'Search ReShade content' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Cinematic effects' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Preview' })).toBeInTheDocument();
  });

  it('resets wizard state when target rendering API changes for the same target', () => {
    const baseTarget = configureSelection(BuildVariant.BuildVariantStandard).target!;
    const { rerender } = render(
      <ReShadeWizard
        catalogue={contentCatalogue()}
        chainTargets={[]}
        gameID={1}
        onClose={vi.fn()}
        onRecoveryRequired={vi.fn()}
        onRefresh={vi.fn()}
        selection={{
          action: Action.ActionConfigureContent,
          candidate: null,
          target: baseTarget,
        }}
      />,
    );

    fireEvent.click(screen.getByRole('checkbox', { name: 'Select Standard effects' }));
    expect(screen.getByRole('checkbox', { name: 'Select Standard effects' })).toBeChecked();

    rerender(
      <ReShadeWizard
        catalogue={contentCatalogue()}
        chainTargets={[]}
        gameID={1}
        onClose={vi.fn()}
        onRecoveryRequired={vi.fn()}
        onRefresh={vi.fn()}
        selection={{
          action: Action.ActionConfigureContent,
          candidate: null,
          target: {
            ...baseTarget,
            ProxyFilename: 'd3d12.dll',
            RenderingAPI: RenderingAPI.RenderingAPID3D12,
          },
        }}
      />,
    );

    expect(screen.getByRole('checkbox', { name: 'Select Standard effects' })).not.toBeChecked();
  });

  it('keeps the content step visible when catalogue props refresh during selection', () => {
    const onClose = vi.fn();
    const { rerender } = render(
      <ReShadeWizard
        catalogue={contentCatalogue()}
        chainTargets={[]}
        gameID={1}
        onClose={onClose}
        onRecoveryRequired={vi.fn()}
        onRefresh={vi.fn()}
        selection={configureSelection(BuildVariant.BuildVariantStandard)}
      />,
    );

    fireEvent.click(screen.getByRole('checkbox', { name: 'Select Standard effects' }));

    rerender(
      <ReShadeWizard
        catalogue={{ ...contentCatalogue(), cached: false }}
        chainTargets={[]}
        gameID={1}
        onClose={onClose}
        onRecoveryRequired={vi.fn()}
        onRefresh={vi.fn()}
        selection={configureSelection(BuildVariant.BuildVariantStandard)}
      />,
    );

    expect(screen.getByRole('searchbox', { name: 'Search ReShade content' })).toBeInTheDocument();
    expect(screen.getByRole('checkbox', { name: 'Select Standard effects' })).toBeChecked();
    expect(onClose).not.toHaveBeenCalled();
  });

  it('refreshes the content catalogue from the search header', async () => {
    vi.mocked(ListReShadeContentCatalogue).mockResolvedValue(contentCatalogue());

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

    fireEvent.click(screen.getByRole('button', { name: 'Refresh catalogue' }));

    await waitFor(() => expect(ListReShadeContentCatalogue).toHaveBeenCalledWith(true));
  });
});
