import type { AppliedProfileSummary } from '@bindings/github.com/phergul/mod-manager/internal/services/models';

import './GameProfilesAppliedSummary.scss';

interface GameProfilesAppliedSummaryProps {
  appliedProfile: AppliedProfileSummary | null;
  isBusy: boolean;
  onRestoreVanilla: () => void;
}

const formatAppliedAt = (appliedAt: string) => {
  if (appliedAt.trim() === '') {
    return 'Applied time unknown';
  }

  const normalizedAppliedAt = appliedAt.includes('T')
    ? appliedAt
    : `${appliedAt.replace(' ', 'T')}Z`;
  const date = new Date(normalizedAppliedAt);
  if (Number.isNaN(date.getTime())) {
    return 'Applied time unknown';
  }

  return `Applied ${new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)}`;
};

export const GameProfilesAppliedSummary = ({
  appliedProfile,
  isBusy,
  onRestoreVanilla,
}: GameProfilesAppliedSummaryProps) => {
  return (
    <div className="game-profiles-applied-summary">
      <div className="game-profiles-applied-summary-copy">
        <span className="game-profiles-applied-summary-label">Applied profile</span>
        <strong className="game-profiles-applied-summary-name">
          {appliedProfile === null ? 'Vanilla' : appliedProfile.ProfileName}
        </strong>
        <span className="game-profiles-applied-summary-meta">
          {appliedProfile === null ? 'No profile applied' : formatAppliedAt(appliedProfile.AppliedAt)}
        </span>
      </div>

      {appliedProfile !== null && (
        <div className="game-profiles-applied-summary-actions">
          <button
            className="game-profiles-applied-summary-button"
            disabled={isBusy}
            onClick={onRestoreVanilla}
            type="button"
          >
            Restore Vanilla
          </button>
        </div>
      )}
    </div>
  );
};
