import { PowerOff } from 'lucide-react';

import type { ModProfile } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

import './GameProfilesActiveSummary.scss';

interface GameProfilesActiveSummaryProps {
  activeProfile: ModProfile | null;
  isBusy: boolean;
  onDeactivateProfile: () => void;
}

export const GameProfilesActiveSummary = ({
  activeProfile,
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
        <button
          className="game-profiles-active-summary-button"
          disabled={isBusy}
          onClick={onDeactivateProfile}
          type="button"
        >
          <PowerOff className="game-profiles-active-summary-button-icon" aria-hidden="true" />
          <span>Deactivate</span>
        </button>
      )}
    </div>
  );
};
