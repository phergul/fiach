import type { FileStateView, StateComparison } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { formatDeploymentBytes, truncateDeploymentHash } from '@utils';

import './DeploymentFourStateView.scss';

const emptyComparison = (): StateComparison => ({
  AppliedMatchesCurrent: false,
  AppliedMatchesDesired: false,
  CurrentMatchesDesired: false,
});

interface DeploymentFourStateViewProps {
  applied: FileStateView | null;
  baseline: FileStateView | null;
  comparison: StateComparison;
  current: FileStateView | null;
  desired: FileStateView | null;
  driftExplanation: string;
  driftKind: string;
  planMode: string;
}

type ColumnKey = 'baseline' | 'applied' | 'current' | 'desired';

interface StateColumnProps {
  columnKey: ColumnKey;
  highlight: 'default' | 'drift' | 'changed' | 'reference' | 'info' | 'missing';
  isDesired?: boolean;
  label: string;
  planMode: string;
  state: FileStateView | null;
}

const isPhaseOneUnavailableState = (
  columnKey: ColumnKey,
  planMode: string,
  state: FileStateView | null,
) => {
  if (planMode !== 'first_apply') {
    return false;
  }

  if (columnKey !== 'baseline' && columnKey !== 'applied') {
    return false;
  }

  if (state === null) {
    return true;
  }

  return !state.Exists && state.Label === '' && state.SHA256 === '' && state.SizeBytes === 0;
};

const isEmptyColumnBody = (columnKey: ColumnKey, planMode: string, state: FileStateView | null) => {
  if (isPhaseOneUnavailableState(columnKey, planMode, state)) {
    return true;
  }

  if (state === null) {
    return true;
  }

  return !state.Exists;
};

const resolveColumnHighlights = (
  comparison: StateComparison,
  driftKind: string,
  planMode: string,
): Record<ColumnKey, StateColumnProps['highlight']> => {
  const isIncremental = planMode === 'incremental';
  const hasDrift = isIncremental && !comparison.AppliedMatchesCurrent;

  return {
    baseline: 'default',
    applied: hasDrift ? 'reference' : 'default',
    current: !isIncremental
      ? 'default'
      : driftKind === 'external'
        ? 'info'
        : driftKind === 'missing'
          ? 'missing'
          : !comparison.AppliedMatchesCurrent
            ? 'drift'
            : 'default',
    desired: !comparison.AppliedMatchesDesired
      ? 'changed'
      : comparison.CurrentMatchesDesired
        ? 'reference'
        : 'default',
  };
};

const StateColumnBody = ({ columnKey, label, planMode, state }: StateColumnProps) => {
  if (isPhaseOneUnavailableState(columnKey, planMode, state)) {
    return <p className="deployment-four-state-view-placeholder">Not available yet</p>;
  }

  if (state === null) {
    return <p className="deployment-four-state-view-placeholder">Not available yet</p>;
  }

  if (!state.Exists) {
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

const StateColumn = ({
  columnKey,
  highlight,
  isDesired = false,
  label,
  planMode,
  state,
}: StateColumnProps) => {
  const isEmpty = isEmptyColumnBody(columnKey, planMode, state);

  return (
    <article
      className={[
        'deployment-four-state-view-column',
        isDesired ? 'deployment-four-state-view-column-desired' : '',
        `deployment-four-state-view-column-${highlight}`,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <header className="deployment-four-state-view-column-header">{label}</header>
      <div
        className={
          isEmpty
            ? 'deployment-four-state-view-column-body deployment-four-state-view-column-body-empty'
            : 'deployment-four-state-view-column-body deployment-four-state-view-column-body-populated'
        }
      >
        <StateColumnBody
          columnKey={columnKey}
          highlight={highlight}
          isDesired={isDesired}
          label={label}
          planMode={planMode}
          state={state}
        />
      </div>
    </article>
  );
};

export const DeploymentFourStateView = ({
  applied,
  baseline,
  comparison,
  current,
  desired,
  driftExplanation,
  driftKind,
  planMode,
}: DeploymentFourStateViewProps) => {
  const safeComparison = comparison ?? emptyComparison();
  const highlights = resolveColumnHighlights(safeComparison, driftKind, planMode);

  return (
    <section className="deployment-four-state-view" aria-label="Four-state comparison">
      <div className="deployment-four-state-view-grid">
        <StateColumn
          columnKey="baseline"
          highlight={highlights.baseline}
          label="Baseline"
          planMode={planMode}
          state={baseline}
        />
        <StateColumn
          columnKey="applied"
          highlight={highlights.applied}
          label="Last applied"
          planMode={planMode}
          state={applied}
        />
        <StateColumn
          columnKey="current"
          highlight={highlights.current}
          label="Current"
          planMode={planMode}
          state={current}
        />
        <StateColumn
          columnKey="desired"
          highlight={highlights.desired}
          isDesired
          label="Desired"
          planMode={planMode}
          state={desired}
        />
      </div>

      {driftExplanation !== '' && (
        <p className="deployment-four-state-view-drift-explanation">{driftExplanation}</p>
      )}
    </section>
  );
};
