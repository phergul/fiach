import { useRef, useState } from 'react';

import { Ellipsis, RefreshCw, ShieldCheck, Trash2, Wrench } from 'lucide-react';

import { Action } from '@bindings/github.com/phergul/fiach/internal/optiscaler/models';
import type {
  OptiScalerCandidate,
  OptiScalerRelease,
  OptiScalerTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';
import { useClickOutside } from '@hooks';
import type { OptiScalerOperationSelection } from '@components/Games/OptiScaler/OptiScalerWizard/OptiScalerWizard';

import './OptiScalerExecutableTable.scss';

interface OptiScalerExecutableTableProps {
  candidates: OptiScalerCandidate[];
  disabled: boolean;
  onStartOperation: (selection: OptiScalerOperationSelection) => void;
  release: OptiScalerRelease | null;
  targets: OptiScalerTarget[];
}

interface ManagedRow {
  target: OptiScalerTarget;
  updateAvailable: boolean;
}

const executableName = (path: string) => path.split(/[\\/]/).pop() ?? path;

const formatGraphicsAPI = (api: string) => {
  switch (api) {
    case 'directx':
      return 'DirectX';
    case 'vulkan':
      return 'Vulkan';
    default:
      return api;
  }
};

const isUpdateAvailable = (target: OptiScalerTarget, release: OptiScalerRelease | null) =>
  release !== null &&
  (target.ReleaseTag !== release.tag ||
    (target.ReleaseDigest !== '' && target.ReleaseDigest !== release.digest));

const managedPriority = (row: ManagedRow) => {
  if (row.target.Status === 'drifted') {
    return 0;
  }
  return row.updateAvailable ? 1 : 2;
};

const statusFacts = (target: OptiScalerTarget, updateAvailable: boolean) => [
  ...(target.Status === 'drifted' ? [{ label: 'Drift detected', tone: 'warning' }] : []),
  ...(updateAvailable ? [{ label: 'Update available', tone: 'info' }] : []),
];

const candidateFacts = (candidate: OptiScalerCandidate) => [
  ...(candidate.hasOptiScaler
    ? [
        { label: 'Files present', tone: 'info' },
        { label: 'OptiScaler', tone: 'success' },
      ]
    : []),
  ...(candidate.hasReShade ? [{ label: 'ReShade', tone: 'warning' }] : []),
];

const configFacts = (target: OptiScalerTarget) => [
  ...(target.DXGISpoofing ? ['DXGI spoof'] : []),
  ...(target.EnableReShadeCoexistence ? ['Loads ReShade'] : []),
];

const meaningfulEvidence = (candidate: OptiScalerCandidate) =>
  candidate.evidence.filter((item) => !item.toLocaleLowerCase().startsWith('validated windows '));

const versionLabel = (target: OptiScalerTarget) => {
  const version = target.ReleaseVersion?.trim() ?? '';
  if (version !== '') {
    return version;
  }
  return 'unknown';
};

const OptiScalerManagedActions = ({
  disabled,
  onStartOperation,
  row,
}: {
  disabled: boolean;
  onStartOperation: (selection: OptiScalerOperationSelection) => void;
  row: ManagedRow;
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const menuAnchorRef = useRef<HTMLDivElement>(null);
  useClickOutside(menuAnchorRef, () => setIsOpen(false), isOpen && !disabled);
  const primaryAction =
    row.target.Status === 'drifted'
      ? Action.ActionRepair
      : row.updateAvailable
        ? Action.ActionUpdate
        : Action.ActionRepair;
  const primaryLabel =
    row.target.Status === 'drifted' ? 'Repair' : row.updateAvailable ? 'Update' : 'Manage';
  const start = (action: Action) => {
    setIsOpen(false);
    onStartOperation({ action, candidate: null, target: row.target });
  };

  return (
    <div className="optiscaler-executable-actions">
      <button
        className="button-main"
        disabled={disabled}
        onClick={() => start(primaryAction)}
        type="button"
      >
        {primaryLabel}
      </button>
      <div className="optiscaler-executable-menu-anchor" ref={menuAnchorRef}>
        <button
          aria-expanded={isOpen}
          aria-label={`More actions for ${executableName(row.target.ExecutableRelativePath)}`}
          disabled={disabled}
          onClick={() => setIsOpen((current) => !current)}
          type="button"
        >
          <Ellipsis aria-hidden="true" />
        </button>
        <DropdownMenu
          ariaLabel={`Actions for ${executableName(row.target.ExecutableRelativePath)}`}
          isOpen={isOpen && !disabled}
          items={[
            {
              disabled: primaryAction === Action.ActionUpdate,
              icon: RefreshCw,
              label: 'Update',
              onSelect: () => start(Action.ActionUpdate),
            },
            {
              disabled: primaryAction === Action.ActionRepair,
              icon: Wrench,
              label: 'Repair',
              onSelect: () => start(Action.ActionRepair),
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

export const OptiScalerExecutableTable = ({
  candidates,
  disabled,
  onStartOperation,
  release,
  targets,
}: OptiScalerExecutableTableProps) => {
  const managedRows = targets
    .map((target) => ({ target, updateAvailable: isUpdateAvailable(target, release) }))
    .sort((left, right) => managedPriority(left) - managedPriority(right));
  const unmanagedCandidates = candidates.filter((candidate) => !candidate.managed);

  return (
    <div className="optiscaler-executable-table">
      <div className="optiscaler-executable-columns" aria-hidden="true">
        <span>Executable</span>
        <span>Details</span>
        <span>State</span>
        <span>Action</span>
      </div>

      <section aria-labelledby="optiscaler-managed-heading">
        <h3 id="optiscaler-managed-heading">Managed</h3>
        {managedRows.length === 0 && (
          <p className="optiscaler-executable-empty">
            No OptiScaler targets are managed for this game.
          </p>
        )}
        {managedRows.map((row) => (
          <div className="optiscaler-executable-row" key={`managed:${row.target.ID}`}>
            <div className="optiscaler-executable-identity">
              <strong>{executableName(row.target.ExecutableRelativePath)}</strong>
              <span>{row.target.TargetRelativePath}</span>
            </div>
            <div className="optiscaler-executable-details">
              <span>{formatGraphicsAPI(row.target.GraphicsAPI)}</span>
              <span>{row.target.ProxyFilename}</span>
              {configFacts(row.target).map((fact) => (
                <span key={fact}>{fact}</span>
              ))}
            </div>
            <div className="optiscaler-executable-state">
              <strong>{versionLabel(row.target)}</strong>
              <div className="optiscaler-executable-status">
                {statusFacts(row.target, row.updateAvailable).length === 0 ? (
                  <span className="optiscaler-executable-status-success">Managed</span>
                ) : (
                  statusFacts(row.target, row.updateAvailable).map((fact) => (
                    <span className={`optiscaler-executable-status-${fact.tone}`} key={fact.label}>
                      {fact.label}
                    </span>
                  ))
                )}
              </div>
            </div>
            <OptiScalerManagedActions
              disabled={disabled}
              onStartOperation={onStartOperation}
              row={row}
            />
          </div>
        ))}
      </section>

      <section aria-labelledby="optiscaler-detected-heading">
        <h3 id="optiscaler-detected-heading">Detected — Not managed</h3>
        {unmanagedCandidates.length === 0 && (
          <p className="optiscaler-executable-empty">
            No unmanaged x64 executable targets were detected.
          </p>
        )}
        {unmanagedCandidates.map((candidate) => {
          const action = candidate.hasOptiScaler ? Action.ActionAdopt : Action.ActionInstall;
          const evidence = meaningfulEvidence(candidate);
          return (
            <div
              className="optiscaler-executable-row"
              key={`${candidate.targetRelativePath}:${candidate.executableRelativePath}`}
            >
              <div className="optiscaler-executable-identity">
                <strong>{candidate.executableName}</strong>
                <span>
                  {candidate.targetRelativePath === '.'
                    ? 'Game Root'
                    : candidate.targetRelativePath}
                </span>
              </div>
              <div className="optiscaler-executable-details">
                <span>{candidate.architecture}</span>
                {evidence.map((item) => (
                  <span key={item}>{item}</span>
                ))}
              </div>
              <div className="optiscaler-executable-state">
                <div className="optiscaler-executable-status">
                  {candidateFacts(candidate).map((fact) => (
                    <span className={`optiscaler-executable-status-${fact.tone}`} key={fact.label}>
                      {fact.label}
                    </span>
                  ))}
                </div>
              </div>
              <div className="optiscaler-executable-actions">
                <button
                  className="button-main"
                  disabled={disabled}
                  onClick={() => onStartOperation({ action, candidate, target: null })}
                  type="button"
                >
                  <ShieldCheck aria-hidden="true" />
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
