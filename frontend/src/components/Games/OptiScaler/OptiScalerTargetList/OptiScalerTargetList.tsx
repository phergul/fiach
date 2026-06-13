import { CheckCircle2, CircleAlert, Cpu, FolderOpen, Sparkles } from 'lucide-react';

import type {
  OptiScalerCandidate,
  OptiScalerRelease,
  OptiScalerTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import './OptiScalerTargetList.scss';

export interface OptiScalerSelection {
  candidate: OptiScalerCandidate | null;
  target: OptiScalerTarget | null;
}

interface OptiScalerTargetListProps {
  candidates: OptiScalerCandidate[];
  disabled: boolean;
  onSelect: (selection: OptiScalerSelection) => void;
  release: OptiScalerRelease | null;
  selectedKey: string | null;
  targets: OptiScalerTarget[];
}

const candidateKey = (candidate: OptiScalerCandidate) =>
  `${candidate.targetRelativePath}:${candidate.executableRelativePath}`;
const targetKey = (target: OptiScalerTarget) => `managed:${target.ID}`;
export const optiScalerSelectionKey = (selection: OptiScalerSelection) =>
  selection.target === null ? candidateKey(selection.candidate!) : targetKey(selection.target);

export const OptiScalerTargetList = ({
  candidates,
  disabled,
  onSelect,
  release,
  selectedKey,
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
            <button
              aria-current={selectedKey === targetKey(target) ? 'true' : undefined}
              className={selectedKey === targetKey(target)
                ? 'optiscaler-target-list-item optiscaler-target-list-item-selected'
                : 'optiscaler-target-list-item'}
              disabled={disabled}
              key={target.ID}
              onClick={() => onSelect({ candidate: null, target })}
              type="button"
            >
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
                  {target.Status === 'drifted' && <li>Drift detected</li>}
                  {updateAvailable && <li>Update available</li>}
                </ul>
              </div>
            </button>
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
          <div className="optiscaler-target-list-group" key={group[0].targetRelativePath}>
            <div className="optiscaler-target-list-item-main">
              <div className="optiscaler-target-list-item-title">
                <FolderOpen aria-hidden="true" />
                <strong>{group[0].targetRelativePath}</strong>
              </div>
              <div className="optiscaler-target-list-candidates">
                {group.map((candidate) => (
                  <button
                    aria-current={selectedKey === candidateKey(candidate) ? 'true' : undefined}
                    className={selectedKey === candidateKey(candidate)
                      ? 'optiscaler-target-list-candidate optiscaler-target-list-item-selected'
                      : 'optiscaler-target-list-candidate'}
                    disabled={disabled}
                    key={candidateKey(candidate)}
                    onClick={() => onSelect({ candidate, target: null })}
                    type="button"
                  >
                    <p>{candidate.executableRelativePath}</p>
                    <ul className="optiscaler-target-list-facts">
                      <li><Cpu aria-hidden="true" /> {candidate.architecture}</li>
                      {candidate.hasOptiScaler && <li><Sparkles aria-hidden="true" /> OptiScaler detected</li>}
                      {candidate.hasReShade && <li>ReShade detected</li>}
                    </ul>
                    <ul className="optiscaler-target-list-evidence" aria-label="Ranking evidence">
                      {candidate.evidence.slice(0, 1).map((evidence) => <li key={evidence}>{evidence}</li>)}
                    </ul>
                  </button>
                ))}
              </div>
            </div>
          </div>
        ))}
      </section>
    </div>
  );
};
