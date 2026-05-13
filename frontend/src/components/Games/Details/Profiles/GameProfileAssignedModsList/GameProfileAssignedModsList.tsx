import type { ProfileMod } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { GameProfileAssignedModRow } from '@components/Games/Details/Profiles/GameProfileAssignedModRow/GameProfileAssignedModRow';

import './GameProfileAssignedModsList.scss';

interface GameProfileAssignedModsListProps {
  isBusy: boolean;
  mods: ProfileMod[];
  onRemoveMod: (modID: number) => void;
  onSetModEnabled: (modID: number, enabled: boolean) => void;
}

export const GameProfileAssignedModsList = ({
  isBusy,
  mods,
  onRemoveMod,
  onSetModEnabled,
}: GameProfileAssignedModsListProps) => {
  return (
    <ul className="game-profile-assigned-mods-list" aria-label="Assigned profile mods">
      {mods.map((mod) => (
        <GameProfileAssignedModRow
          key={mod.ModID}
          isBusy={isBusy}
          mod={mod}
          onRemoveMod={onRemoveMod}
          onSetModEnabled={onSetModEnabled}
        />
      ))}
    </ul>
  );
};
