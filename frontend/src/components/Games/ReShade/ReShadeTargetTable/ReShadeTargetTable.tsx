import { useMemo, useRef, useState } from 'react';

import {
  Ellipsis,
  PackagePlus,
  RefreshCw,
  ShieldCheck,
  SlidersHorizontal,
  Trash2,
  Wrench,
} from 'lucide-react';

import { Action } from '@bindings/github.com/phergul/fiach/internal/reshade/models';
import type {
  ReShadeChainTarget,
  ReShadeDiscoveryResult,
  ReShadeInstallerStatus,
  ReShadeTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';
import { isReShadeUpdateAvailable, useClickOutside } from '@hooks';

import './ReShadeTargetTable.scss';

export interface ReShadeOperationSelection {
  action: Action;
  candidate: ReShadeDiscoveryResult['candidates'][number] | null;
  target: ReShadeTarget | null;
}

interface ReShadeTargetTableProps {
  chainTargets: ReShadeChainTarget[];
  disabled: boolean;
  discovery: ReShadeDiscoveryResult | null;
  installerStatus: ReShadeInstallerStatus | null;
  onStartOperation: (selection: ReShadeOperationSelection) => void;
  targets: ReShadeTarget[];
}

const filename = (path: string) => path.split(/[\\/]/).pop() ?? path;

const targetKey = (path: string) => path.trim().toLocaleLowerCase();

const hasDetectedReShade = (candidate: ReShadeDiscoveryResult['candidates'][number]) =>
  candidate.proxyEvidence.some((evidence) => evidence.isReShade);

const formatRenderingAPI = (api: string) => {
  switch (api) {
    case 'd3d9':
      return 'D3D9';
    case 'd3d10':
      return 'D3D10';
    case 'd3d11':
      return 'D3D11';
    case 'd3d12':
      return 'D3D12';
    case 'opengl':
      return 'OpenGL';
    default:
      return api;
  }
};

const formatRuntimeVersion = (version: string | null | undefined) => {
  const trimmed = version?.trim() ?? '';
  if (trimmed === '') {
    return '';
  }
  return trimmed.toLowerCase().startsWith('v') ? trimmed : `v${trimmed}`;
};

const detectedRuntimeVersions = (candidate: ReShadeDiscoveryResult['candidates'][number]) => [
  ...new Set(
    candidate.proxyEvidence
      .filter((evidence) => evidence.isReShade)
      .map((evidence) => formatRuntimeVersion(evidence.runtimeVersion))
      .filter((version) => version !== ''),
  ),
];

const detectedProxyEvidence = (candidate: ReShadeDiscoveryResult['candidates'][number]) =>
  candidate.proxyEvidence
    .filter((evidence) => evidence.exists)
    .map((evidence) => {
      if (evidence.isReShade) {
        const version = formatRuntimeVersion(evidence.runtimeVersion);
        return version === '' ? evidence.filename : `${evidence.filename} ${version}`;
      }
      return evidence.conflict?.trim() ? `${evidence.filename} conflict` : evidence.filename;
    });

const chainSummary = (chainTargets: ReShadeChainTarget[], targetRelativePath: string) => {
  const chain = chainTargets.find(
    (item) => targetKey(item.TargetRelativePath) === targetKey(targetRelativePath),
  );
  if (chain === undefined) {
    return 'No managed chain';
  }
  if (chain.OptiScaler !== null && chain.ReShade !== null) {
    return `OptiScaler primary · ${chain.PrimaryProxyFilename}`;
  }
  return `${chain.PrimaryOwner} primary · ${chain.PrimaryProxyFilename}`;
};

const managedFacts = (target: ReShadeTarget, installerStatus: ReShadeInstallerStatus | null) => {
  return [
    {
      label: target.Status === 'drifted' ? 'Drift detected' : target.Status,
      tone: target.Status === 'drifted' ? 'warning' : 'success',
    },
    ...(isReShadeUpdateAvailable(target, installerStatus)
      ? [{ label: 'Update available', tone: 'info' }]
      : []),
  ];
};

const variantLabel = (target: ReShadeTarget) =>
  target.BuildVariant === 'addon' ? 'Full add-on' : 'Standard';

const provenanceLabel = (target: ReShadeTarget) =>
  target.VariantProvenance === 'user_declared' ? 'User declared' : 'Verified';

const activeRuntimeLabel = (target: ReShadeTarget) =>
  target.ActiveRuntimeFilename.trim() === '' ? target.ProxyFilename : target.ActiveRuntimeFilename;

const managedDetailFacts = (target: ReShadeTarget, chainTargets: ReShadeChainTarget[]) => [
  formatRenderingAPI(target.RenderingAPI),
  target.Architecture,
  variantLabel(target),
  provenanceLabel(target),
  target.ProxyFilename,
  ...(activeRuntimeLabel(target) === target.ProxyFilename ? [] : [activeRuntimeLabel(target)]),
  chainSummary(chainTargets, target.TargetRelativePath),
];

const ReShadeManagedActions = ({
  disabled,
  onStartOperation,
  target,
  updateAvailable,
}: {
  disabled: boolean;
  onStartOperation: (selection: ReShadeOperationSelection) => void;
  target: ReShadeTarget;
  updateAvailable: boolean;
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const menuAnchorRef = useRef<HTMLDivElement>(null);
  useClickOutside(menuAnchorRef, () => setIsOpen(false), isOpen && !disabled);
  const primaryAction =
    target.Status === 'drifted'
      ? Action.ActionRepair
      : updateAvailable
        ? Action.ActionUpdate
        : Action.ActionConfigureContent;
  const primaryLabel =
    target.Status === 'drifted' ? 'Repair' : updateAvailable ? 'Update' : 'Content';
  const start = (action: Action) => {
    setIsOpen(false);
    onStartOperation({ action, candidate: null, target });
  };

  return (
    <div className="reshade-target-actions">
      <button
        className="button-main"
        disabled={disabled}
        onClick={() => start(primaryAction)}
        type="button"
      >
        {primaryLabel}
      </button>
      <div className="reshade-target-menu-anchor" ref={menuAnchorRef}>
        <button
          aria-expanded={isOpen}
          aria-label={`More actions for ${filename(target.ExecutableRelativePath)}`}
          disabled={disabled}
          onClick={() => setIsOpen((current) => !current)}
          type="button"
        >
          <Ellipsis aria-hidden="true" />
        </button>
        <DropdownMenu
          ariaLabel={`Actions for ${filename(target.ExecutableRelativePath)}`}
          isOpen={isOpen && !disabled}
          items={[
            {
              icon: RefreshCw,
              label: 'Update',
              onSelect: () => start(Action.ActionUpdate),
            },
            {
              icon: Wrench,
              label: 'Repair',
              onSelect: () => start(Action.ActionRepair),
            },
            {
              icon: SlidersHorizontal,
              label: 'Configure content',
              onSelect: () => start(Action.ActionConfigureContent),
            },
            {
              icon: Trash2,
              label: 'Uninstall',
              onSelect: () => start(Action.ActionUninstall),
            },
          ]}
        />
      </div>
    </div>
  );
};

export const ReShadeTargetTable = ({
  chainTargets,
  disabled,
  discovery,
  installerStatus,
  onStartOperation,
  targets,
}: ReShadeTargetTableProps) => {
  const managedKeys = useMemo(
    () => new Set(targets.map((target) => targetKey(target.TargetRelativePath))),
    [targets],
  );
  const detected =
    discovery?.candidates.filter(
      (candidate) => !managedKeys.has(targetKey(candidate.targetRelativePath)),
    ) ?? [];

  return (
    <div className="reshade-target-table">
      <div className="reshade-target-columns" aria-hidden="true">
        <span>Executable</span>
        <span>Details</span>
        <span>State</span>
        <span>Action</span>
      </div>

      <section aria-labelledby="reshade-managed-heading">
        <h3 id="reshade-managed-heading">Managed</h3>
        {targets.length === 0 && (
          <p className="reshade-target-empty">No ReShade targets are managed for this game.</p>
        )}
        {targets.map((target) => {
          const updateAvailable = isReShadeUpdateAvailable(target, installerStatus);
          return (
            <div className="reshade-target-row" key={`managed:${target.ID}`}>
              <div className="reshade-target-identity">
                <strong>{filename(target.ExecutableRelativePath)}</strong>
                <span>
                  {target.TargetRelativePath === '.' ? 'Game Root' : target.TargetRelativePath}
                </span>
              </div>
              <div className="reshade-target-details">
                {managedDetailFacts(target, chainTargets).map((fact) => (
                  <span key={fact}>{fact}</span>
                ))}
              </div>
              <div className="reshade-target-state">
                <strong>{formatRuntimeVersion(target.RuntimeVersion) || 'Unknown runtime'}</strong>
                <div className="reshade-target-status">
                  {managedFacts(target, installerStatus).map((fact) => (
                    <span className={`reshade-target-status-${fact.tone}`} key={fact.label}>
                      {fact.label}
                    </span>
                  ))}
                </div>
              </div>
              <ReShadeManagedActions
                disabled={disabled}
                onStartOperation={onStartOperation}
                target={target}
                updateAvailable={updateAvailable}
              />
            </div>
          );
        })}
      </section>

      <section aria-labelledby="reshade-detected-heading">
        <h3 id="reshade-detected-heading">Detected - Not managed</h3>
        {detected.length === 0 && (
          <p className="reshade-target-empty">
            No unmanaged ReShade executable targets were detected.
          </p>
        )}
        {detected.map((candidate) => {
          const action = hasDetectedReShade(candidate) ? Action.ActionAdopt : Action.ActionInstall;
          const runtimeVersions = detectedRuntimeVersions(candidate);
          const proxyEvidence = detectedProxyEvidence(candidate);
          return (
            <div
              className="reshade-target-row"
              key={`${candidate.targetRelativePath}:${candidate.executableRelativePath}`}
            >
              <div className="reshade-target-identity">
                <strong>{filename(candidate.executableRelativePath)}</strong>
                <span>
                  {candidate.targetRelativePath === '.'
                    ? 'Game Root'
                    : candidate.targetRelativePath}
                </span>
              </div>
              <div className="reshade-target-details">
                <span>
                  {candidate.apiOptions
                    .map((option) => formatRenderingAPI(option.renderingApi))
                    .join(', ')}
                </span>
                <span>{candidate.architecture}</span>
                {runtimeVersions.length === 0
                  ? null
                  : runtimeVersions.map((version) => <span key={version}>Runtime {version}</span>)}
                {proxyEvidence.map((evidence) => (
                  <span key={evidence}>{evidence}</span>
                ))}
              </div>
              <div className="reshade-target-state">
                <div className="reshade-target-status">
                  {candidate.conflicts.length > 0 && (
                    <span className="reshade-target-status-warning">Conflict</span>
                  )}
                  {hasDetectedReShade(candidate) && (
                    <span className="reshade-target-status-info">ReShade</span>
                  )}
                </div>
              </div>
              <div className="reshade-target-actions">
                <button
                  className="button-main"
                  disabled={disabled || candidate.conflicts.length > 0}
                  onClick={() => onStartOperation({ action, candidate, target: null })}
                  type="button"
                >
                  {action === Action.ActionAdopt ? (
                    <ShieldCheck aria-hidden="true" />
                  ) : (
                    <PackagePlus aria-hidden="true" />
                  )}
                  {action === Action.ActionAdopt ? 'Adopt' : 'Install'}
                </button>
              </div>
            </div>
          );
        })}
      </section>
    </div>
  );
};
