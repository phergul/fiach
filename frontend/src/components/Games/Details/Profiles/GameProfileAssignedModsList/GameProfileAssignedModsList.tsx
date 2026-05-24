import type { ProfileMod } from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';
import { GameProfileAssignedModRow } from '@components/Games/Details/Profiles/GameProfileAssignedModRow/GameProfileAssignedModRow';

import './GameProfileAssignedModsList.scss';

interface GameProfileAssignedModsListProps {
  isBusy: boolean;
  mods: ProfileMod[];
  onMoveMod: (modID: number, direction: -1 | 1) => void;
  onRemoveMod: (modID: number) => void;
  onSetModEnabled: (modID: number, enabled: boolean) => void;
}

export const GameProfileAssignedModsList = ({
  isBusy,
  mods,
  onMoveMod,
  onRemoveMod,
  onSetModEnabled,
}: GameProfileAssignedModsListProps) => {
  return (
    <ul className="game-profile-assigned-mods-list" aria-label="Assigned profile mods">
      {mods.map((mod, index) => (
        <GameProfileAssignedModRow
          key={mod.ModID}
          canMoveDown={index < mods.length - 1}
          canMoveUp={index > 0}
          isBusy={isBusy}
          mod={mod}
          onMoveDown={() => onMoveMod(mod.ModID, 1)}
          onMoveUp={() => onMoveMod(mod.ModID, -1)}
          onRemoveMod={onRemoveMod}
          onSetModEnabled={onSetModEnabled}
        />
      ))}
    </ul>
  );
};
