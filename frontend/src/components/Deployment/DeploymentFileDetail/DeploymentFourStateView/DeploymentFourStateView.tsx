import type { FileStateView } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { formatDeploymentBytes, truncateDeploymentHash } from '@utils';

import './DeploymentFourStateView.scss';

interface DeploymentFourStateViewProps {
  baseline: FileStateView | null;
  applied: FileStateView | null;
  current: FileStateView | null;
  desired: FileStateView | null;
}

interface StateColumnProps {
  isDesired?: boolean;
  label: string;
  state: FileStateView | null;
}

const isPhaseOneUnavailableState = (label: string, state: FileStateView | null) => {
  if (label !== 'Baseline' && label !== 'Last applied') {
    return false;
  }

  if (state === null) {
    return true;
  }

  return !state.Exists && state.Label === '' && state.SHA256 === '' && state.SizeBytes === 0;
};

const isEmptyColumnBody = (label: string, state: FileStateView | null) => {
  if (isPhaseOneUnavailableState(label, state)) {
    return true;
  }

  if (state === null) {
    return true;
  }

  return !state.Exists;
};

const StateColumnBody = ({ label, state }: StateColumnProps) => {
  if (isPhaseOneUnavailableState(label, state)) {
    return <p className="deployment-four-state-view-placeholder">Not available yet</p>;
  }

  const isUnavailable = state === null || !state.Exists;

  if (state === null) {
    return <p className="deployment-four-state-view-placeholder">Not available yet</p>;
  }

  if (isUnavailable) {
    return <p className="deployment-four-state-view-empty">{state.Label || 'Not present'}</p>;
  }

  return (
    <dl className="deployment-four-state-view-details">
      <div className="deployment-four-state-view-detail">
        <dt>Label</dt>
        <dd>{state.Label}</dd>
      </div>
      <div className="deployment-four-state-view-detail">
        <dt>Size</dt>
        <dd>{formatDeploymentBytes(state.SizeBytes)}</dd>
      </div>
      {state.SHA256 !== '' && (
        <div className="deployment-four-state-view-detail">
          <dt>SHA256</dt>
          <dd className="deployment-four-state-view-hash" title={state.SHA256}>
            {truncateDeploymentHash(state.SHA256)}
          </dd>
        </div>
      )}
    </dl>
  );
};

const StateColumn = ({ isDesired = false, label, state }: StateColumnProps) => {
  const isEmpty = isEmptyColumnBody(label, state);

  return (
    <article
      className={
        isDesired
          ? 'deployment-four-state-view-column deployment-four-state-view-column-desired'
          : 'deployment-four-state-view-column'
      }
    >
      <header className="deployment-four-state-view-column-header">{label}</header>
      <div
        className={
          isEmpty
            ? 'deployment-four-state-view-column-body deployment-four-state-view-column-body-empty'
            : 'deployment-four-state-view-column-body deployment-four-state-view-column-body-populated'
        }
      >
        <StateColumnBody label={label} state={state} />
      </div>
    </article>
  );
};

export const DeploymentFourStateView = ({
  applied,
  baseline,
  current,
  desired,
}: DeploymentFourStateViewProps) => {
  return (
    <section className="deployment-four-state-view" aria-label="Four-state comparison">
      <StateColumn label="Baseline" state={baseline} />
      <StateColumn label="Last applied" state={applied} />
      <StateColumn label="Current" state={current} />
      <StateColumn isDesired label="Desired" state={desired} />
    </section>
  );
};
