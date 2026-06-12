import { ListFilter } from 'lucide-react';

import './GameProfileModsFilter.scss';

interface GameProfileModsFilterProps {
  enabledCount: number;
  isEnabledOnly: boolean;
  totalCount: number;
  onEnabledOnlyChange: (isEnabledOnly: boolean) => void;
}

export const GameProfileModsFilter = ({
  enabledCount,
  isEnabledOnly,
  totalCount,
  onEnabledOnlyChange,
}: GameProfileModsFilterProps) => {
  return (
    <button
      aria-pressed={isEnabledOnly}
      className={
        isEnabledOnly
          ? 'game-profile-mods-filter game-profile-mods-filter-active'
          : 'game-profile-mods-filter'
      }
      onClick={() => onEnabledOnlyChange(!isEnabledOnly)}
      type="button"
    >
      <ListFilter className="game-profile-mods-filter-icon" aria-hidden="true" />
      Enabled only ({enabledCount}/{totalCount})
    </button>
  );
};
