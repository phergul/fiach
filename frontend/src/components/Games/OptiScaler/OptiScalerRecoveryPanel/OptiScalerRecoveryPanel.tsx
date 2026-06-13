import type { OptiScalerRecoveryState } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './OptiScalerRecoveryPanel.scss';

interface OptiScalerRecoveryPanelProps {
  isRollingBack: boolean;
  onRollback: () => void;
  recovery: OptiScalerRecoveryState;
}

export const OptiScalerRecoveryPanel = ({
  isRollingBack,
  onRollback,
  recovery,
}: OptiScalerRecoveryPanelProps) => {
  return (
    <section className="optiscaler-recovery-panel" aria-label="OptiScaler recovery required">
      <div>
        <h2 className="optiscaler-recovery-panel-title">Recovery required</h2>
        <p className="optiscaler-recovery-panel-message">
          An incomplete {recovery.action ?? 'OptiScaler'} operation is blocking lifecycle changes.
        </p>
        <dl className="optiscaler-recovery-panel-details">
          <div>
            <dt>Game ID</dt>
            <dd>{recovery.gameId ?? '-'}</dd>
          </div>
          <div>
            <dt>Target</dt>
            <dd>{recovery.targetPath ?? '-'}</dd>
          </div>
          {recovery.error !== undefined && recovery.error !== '' && (
            <div>
              <dt>Error</dt>
              <dd>{recovery.error}</dd>
            </div>
          )}
        </dl>
      </div>
      <button
        className="optiscaler-recovery-panel-action"
        disabled={isRollingBack}
        onClick={onRollback}
        type="button"
      >
        {isRollingBack ? 'Rolling back...' : 'Roll back operation'}
      </button>
    </section>
  );
};
