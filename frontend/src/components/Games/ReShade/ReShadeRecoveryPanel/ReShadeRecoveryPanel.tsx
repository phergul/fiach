import type { ReShadeRecoveryState } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './ReShadeRecoveryPanel.scss';

interface ReShadeRecoveryPanelProps {
  isRollingBack: boolean;
  onRollback: () => void;
  recovery: ReShadeRecoveryState;
}

export const ReShadeRecoveryPanel = ({
  isRollingBack,
  onRollback,
  recovery,
}: ReShadeRecoveryPanelProps) => (
  <section className="reshade-recovery-panel" aria-label="ReShade recovery">
    <div>
      <h3>Recovery required</h3>
      <p>{recovery.error ?? 'A managed ReShade operation stopped before cleanup completed.'}</p>
    </div>
    <button className="button-warning" disabled={isRollingBack || recovery.journalId === undefined} onClick={onRollback} type="button">
      {isRollingBack ? 'Rolling back' : 'Rollback'}
    </button>
  </section>
);
