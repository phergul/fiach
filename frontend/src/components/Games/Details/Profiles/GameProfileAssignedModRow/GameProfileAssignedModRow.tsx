import { Trash2 } from 'lucide-react';

import type { ProfileMod } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

import './GameProfileAssignedModRow.scss';

interface GameProfileAssignedModRowProps {
  isBusy: boolean;
  mod: ProfileMod;
  onRemoveMod: (modID: number) => void;
  onSetModEnabled: (modID: number, enabled: boolean) => void;
}

export const GameProfileAssignedModRow = ({
  isBusy,
  mod,
  onRemoveMod,
  onSetModEnabled,
}: GameProfileAssignedModRowProps) => {
  const displayLoadOrder = mod.LoadOrder + 1;

  return (
    <li className="game-profile-assigned-mod-row">
      <div className="game-profile-assigned-mod-row-order" aria-label={`Load order ${displayLoadOrder}`}>
        {displayLoadOrder}
      </div>

      <div className="game-profile-assigned-mod-row-main">
        <span className="game-profile-assigned-mod-row-name">{mod.Name}</span>
      </div>

      <div className="game-profile-assigned-mod-row-actions">
        <label className="game-profile-assigned-mod-row-toggle">
          <input
            checked={mod.Enabled}
            disabled={isBusy}
            onChange={(event) => onSetModEnabled(mod.ModID, event.target.checked)}
            type="checkbox"
          />
          <span className="game-profile-assigned-mod-row-toggle-control" aria-hidden="true" />
        </label>

        <button
          className="game-profile-assigned-mod-row-icon-button game-profile-assigned-mod-row-icon-button-danger"
          disabled={isBusy}
          onClick={() => onRemoveMod(mod.ModID)}
          title="Remove mod from profile"
          type="button"
        >
          <Trash2 className="game-profile-assigned-mod-row-icon" aria-hidden="true" />
        </button>
      </div>
    </li>
  );
};
