import { Link } from 'react-router-dom';

import type { ModProfile } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

import './GameProfilesActiveSummary.scss';

interface GameProfilesActiveSummaryProps {
  activeProfile: ModProfile | null;
  applyProfilePath: string;
  isBusy: boolean;
  onDeactivateProfile: () => void;
}

export const GameProfilesActiveSummary = ({
  activeProfile,
  applyProfilePath,
  isBusy,
  onDeactivateProfile,
}: GameProfilesActiveSummaryProps) => {
  return (
    <div className="game-profiles-active-summary">
      <div className="game-profiles-active-summary-copy">
        <span className="game-profiles-active-summary-label">Active profile</span>
        <strong className="game-profiles-active-summary-name">
          {activeProfile === null ? 'No active profile' : activeProfile.Name}
        </strong>
      </div>
      {activeProfile !== null && (
        <div className="game-profiles-active-summary-actions">
          <Link
            className={
              isBusy
                ? 'game-profiles-active-summary-button game-profiles-active-summary-apply game-profiles-active-summary-link-disabled'
                : 'game-profiles-active-summary-button game-profiles-active-summary-apply'
            }
            to={applyProfilePath}
            onClick={(event) => {
              if (isBusy) {
                event.preventDefault();
              }
            }}
            aria-disabled={isBusy}
          >
            Apply Profile
          </Link>
          <button
            className="game-profiles-active-summary-button"
            disabled={isBusy}
            onClick={onDeactivateProfile}
            type="button"
          >
            <span>Deactivate</span>
          </button>
        </div>
      )}
    </div>
  );
};
