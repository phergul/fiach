import { CheckCircle2, CircleAlert, Cpu, FolderOpen, Sparkles } from 'lucide-react';

import type {
  OptiScalerCandidate,
  OptiScalerRelease,
  OptiScalerTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { Action } from '@bindings/github.com/phergul/fiach/internal/optiscaler/models';

import './OptiScalerTargetList.scss';

export interface OptiScalerSelection {
  action: Action;
  candidate: OptiScalerCandidate | null;
  target: OptiScalerTarget | null;
}

interface OptiScalerTargetListProps {
  candidates: OptiScalerCandidate[];
  disabled: boolean;
  onSelect: (selection: OptiScalerSelection) => void;
  release: OptiScalerRelease | null;
  targets: OptiScalerTarget[];
}

const candidateKey = (candidate: OptiScalerCandidate) =>
  `${candidate.targetRelativePath}:${candidate.executableRelativePath}`;

export const OptiScalerTargetList = ({
  candidates,
  disabled,
  onSelect,
  release,
  targets,
}: OptiScalerTargetListProps) => {
  const unmanagedCandidates = candidates.filter((candidate) => !candidate.managed);
  const unmanagedGroups = Object.values(
    unmanagedCandidates.reduce<Record<string, OptiScalerCandidate[]>>((groups, candidate) => {
      const key = candidate.targetRelativePath.toLowerCase();
      groups[key] = [...(groups[key] ?? []), candidate];
      return groups;
    }, {}),
  );

  return (
    <div className="optiscaler-target-list">
      <section className="optiscaler-target-list-section" aria-labelledby="optiscaler-managed-title">
        <div className="optiscaler-target-list-heading">
          <h2 id="optiscaler-managed-title">Managed targets</h2>
          <span>{targets.length}</span>
        </div>
        {targets.length === 0 && (
          <p className="optiscaler-target-list-empty">No OptiScaler targets are managed for this game.</p>
        )}
        {targets.map((target) => {
          const updateAvailable = release !== null && (
            target.ReleaseTag !== release.tag ||
            (target.ReleaseDigest !== '' && target.ReleaseDigest !== release.digest)
          );
          return (
            <article className="optiscaler-target-list-item" key={target.ID}>
              <div className="optiscaler-target-list-item-main">
                <div className="optiscaler-target-list-item-title">
                  {target.Status === 'drifted' ? (
                    <CircleAlert aria-hidden="true" />
                  ) : (
                    <CheckCircle2 aria-hidden="true" />
                  )}
                  <strong>{target.TargetRelativePath}</strong>
                </div>
                <p>{target.ExecutableRelativePath}</p>
                <ul className="optiscaler-target-list-facts">
                  <li>{target.GraphicsAPI === 'vulkan' ? 'Vulkan' : 'DirectX'}</li>
                  <li>{target.ProxyFilename}</li>
                  <li>{target.ReleaseVersion || target.ReleaseTag}</li>
                  <li>{target.ManagementOrigin === 'adopted' ? 'Adopted' : 'Installed by Fiach'}</li>
                  {target.Status === 'drifted' && <li>Drift detected</li>}
                  {updateAvailable && <li>Update available</li>}
                </ul>
              </div>
              <div className="optiscaler-target-list-actions">
                <button
                  disabled={disabled}
                  onClick={() => onSelect({ action: Action.ActionUpdate, candidate: null, target })}
                  type="button"
                >
                  Update
                </button>
                <button
                  disabled={disabled}
                  onClick={() => onSelect({ action: Action.ActionRepair, candidate: null, target })}
                  type="button"
                >
                  Repair
                </button>
                <button
                  disabled={disabled}
                  onClick={() => onSelect({ action: Action.ActionUninstall, candidate: null, target })}
                  type="button"
                >
                  Uninstall
                </button>
              </div>
            </article>
          );
        })}
      </section>

      <section className="optiscaler-target-list-section" aria-labelledby="optiscaler-detected-title">
        <div className="optiscaler-target-list-heading">
          <h2 id="optiscaler-detected-title">Detected unmanaged targets</h2>
          <span>{unmanagedGroups.length}</span>
        </div>
        {unmanagedGroups.length === 0 && (
          <p className="optiscaler-target-list-empty">No unmanaged x64 executable targets were detected.</p>
        )}
        {unmanagedGroups.map((group) => (
          <article className="optiscaler-target-list-item" key={group[0].targetRelativePath}>
            <div className="optiscaler-target-list-item-main">
              <div className="optiscaler-target-list-item-title">
                <FolderOpen aria-hidden="true" />
                <strong>{group[0].targetRelativePath}</strong>
              </div>
              <div className="optiscaler-target-list-candidates">
                {group.map((candidate) => (
                  <div className="optiscaler-target-list-candidate" key={candidateKey(candidate)}>
                    <p>{candidate.executableRelativePath}</p>
                    <ul className="optiscaler-target-list-facts">
                      <li><Cpu aria-hidden="true" /> {candidate.architecture}</li>
                      {candidate.hasOptiScaler && <li><Sparkles aria-hidden="true" /> OptiScaler detected</li>}
                      {candidate.hasReShade && <li>ReShade detected</li>}
                    </ul>
                    <ul className="optiscaler-target-list-evidence" aria-label="Ranking evidence">
                      {candidate.evidence.map((evidence) => <li key={evidence}>{evidence}</li>)}
                    </ul>
                    <div className="optiscaler-target-list-actions">
                      <button
                        disabled={disabled || candidate.hasOptiScaler}
                        onClick={() => onSelect({ action: Action.ActionInstall, candidate, target: null })}
                        type="button"
                      >
                        Install
                      </button>
                      <button
                        disabled={disabled || !candidate.hasOptiScaler}
                        onClick={() => onSelect({ action: Action.ActionAdopt, candidate, target: null })}
                        type="button"
                      >
                        Adopt
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </article>
        ))}
      </section>
    </div>
  );
};
