import { CircleAlert, Cpu, FolderOpen, Gauge, ShieldAlert } from 'lucide-react';

import { Action } from '@bindings/github.com/phergul/fiach/internal/optiscaler/models';
import type {
  OptiScalerCandidate,
  OptiScalerRelease,
  OptiScalerTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import type { OptiScalerSelection } from '@components/Games/OptiScaler/OptiScalerTargetList/OptiScalerTargetList';

import './OptiScalerDetail.scss';

interface OptiScalerDetailProps {
  candidateCount: number;
  managedCount: number;
  onStartAction: (action: Action) => void;
  release: OptiScalerRelease | null;
  selection: OptiScalerSelection | null;
}

const candidateDetails = (candidate: OptiScalerCandidate) => [
  { label: 'Directory', value: candidate.targetRelativePath },
  { label: 'Executable', value: candidate.executableRelativePath },
  { label: 'Architecture', value: candidate.architecture },
  { label: 'OptiScaler', value: candidate.hasOptiScaler ? 'Detected' : 'Not detected' },
  { label: 'ReShade', value: candidate.hasReShade ? 'Detected' : 'Not detected' },
];

const managedDetails = (target: OptiScalerTarget) => [
  { label: 'Directory', value: target.TargetRelativePath },
  { label: 'Executable', value: target.ExecutableRelativePath },
  { label: 'Graphics API', value: target.GraphicsAPI === 'vulkan' ? 'Vulkan' : 'DirectX' },
  { label: 'Proxy', value: target.ProxyFilename },
  { label: 'Release', value: target.ReleaseVersion || target.ReleaseTag },
  { label: 'Status', value: target.Status },
];

export const OptiScalerDetail = ({
  candidateCount,
  managedCount,
  onStartAction,
  release,
  selection,
}: OptiScalerDetailProps) => {
  if (selection === null) {
    return (
      <div className="optiscaler-detail optiscaler-detail-overview">
        <div className="optiscaler-detail-header">
          <div>
            <h2>OptiScaler overview</h2>
            <p>Select a managed target or detected executable to review available actions.</p>
          </div>
        </div>
        <div className="optiscaler-detail-overview-grid">
          <dl>
            <div><dt>Stable release</dt><dd>{release?.version || release?.tag || 'Unavailable'}</dd></div>
            <div><dt>Managed targets</dt><dd>{managedCount}</dd></div>
            <div><dt>Detected executables</dt><dd>{candidateCount}</dd></div>
          </dl>
          <div className="optiscaler-detail-boundaries">
            <p><Gauge aria-hidden="true" /> DirectX recommends <strong>dxgi.dll</strong>.</p>
            <p><Cpu aria-hidden="true" /> Vulkan recommends <strong>winmm.dll</strong>.</p>
            <p><ShieldAlert aria-hidden="true" /> Online games and anti-cheat remain game-specific risks.</p>
            <p><CircleAlert aria-hidden="true" /> Vulkan and ReShade coexistence is not automated.</p>
          </div>
        </div>
      </div>
    );
  }

  const candidate = selection.candidate;
  const target = selection.target;
  if (candidate === null && target === null) {
    return null;
  }

  const details = candidate !== null
    ? candidateDetails(candidate)
    : managedDetails(target as OptiScalerTarget);
  const title = candidate?.executableName ?? target?.ExecutableRelativePath.split(/[\\/]/).pop() ?? 'Target';

  return (
    <div className="optiscaler-detail">
      <div className="optiscaler-detail-header">
        <div>
          <h2>{title}</h2>
          <p>{candidate?.targetRelativePath ?? target?.TargetRelativePath}</p>
        </div>
        <div className="optiscaler-detail-actions">
          {candidate !== null ? (
            <button
              className="button-main"
              onClick={() => onStartAction(candidate.hasOptiScaler ? Action.ActionAdopt : Action.ActionInstall)}
              type="button"
            >
              {candidate.hasOptiScaler ? 'Adopt' : 'Install'}
            </button>
          ) : (
            <>
              <button onClick={() => onStartAction(Action.ActionUpdate)} type="button">Update</button>
              <button onClick={() => onStartAction(Action.ActionRepair)} type="button">Repair</button>
              <button onClick={() => onStartAction(Action.ActionUninstall)} type="button">Uninstall</button>
            </>
          )}
        </div>
      </div>
      <div className="optiscaler-detail-body">
        <dl className="optiscaler-detail-properties">
          {details.map((detail) => (
            <div key={detail.label}>
              <dt>{detail.label}</dt>
              <dd>{detail.value}</dd>
            </div>
          ))}
        </dl>
        {candidate !== null && candidate.evidence.length > 0 && (
          <section className="optiscaler-detail-evidence">
            <h3>Ranking evidence</h3>
            <ul>{candidate.evidence.map((evidence) => <li key={evidence}>{evidence}</li>)}</ul>
          </section>
        )}
        <p className="optiscaler-detail-note">
          <FolderOpen aria-hidden="true" />
          Target ranking helps identify likely game executables but does not guarantee compatibility.
        </p>
      </div>
    </div>
  );
};
