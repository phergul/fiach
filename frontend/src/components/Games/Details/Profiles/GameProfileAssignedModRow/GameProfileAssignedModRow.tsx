import { ArrowDown, ArrowUp, Trash2 } from 'lucide-react';

import type { ProfileMod } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

import './GameProfileAssignedModRow.scss';

interface GameProfileAssignedModRowProps {
  canMoveDown: boolean;
  canMoveUp: boolean;
  isBusy: boolean;
  mod: ProfileMod;
  onMoveDown: () => void;
  onMoveUp: () => void;
  onRemoveMod: (modID: number) => void;
  onSetModEnabled: (modID: number, enabled: boolean) => void;
}

export const GameProfileAssignedModRow = ({
  canMoveDown,
  canMoveUp,
  isBusy,
  mod,
  onMoveDown,
  onMoveUp,
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
        <div className="game-profile-assigned-mod-row-order-actions">
          <button
            aria-label={`Move ${mod.Name} up`}
            className="game-profile-assigned-mod-row-icon-button"
            disabled={isBusy || !canMoveUp}
            onClick={onMoveUp}
            title="Move mod up"
            type="button"
          >
            <ArrowUp className="game-profile-assigned-mod-row-icon" aria-hidden="true" />
          </button>

          <button
            aria-label={`Move ${mod.Name} down`}
            className="game-profile-assigned-mod-row-icon-button"
            disabled={isBusy || !canMoveDown}
            onClick={onMoveDown}
            title="Move mod down"
            type="button"
          >
            <ArrowDown className="game-profile-assigned-mod-row-icon" aria-hidden="true" />
          </button>
        </div>

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
          aria-label={`Remove ${mod.Name} from profile`}
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
