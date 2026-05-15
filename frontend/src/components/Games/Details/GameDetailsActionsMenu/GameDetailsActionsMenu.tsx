import { FolderCog, RotateCcw } from 'lucide-react';

import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

import './GameDetailsActionsMenu.scss';

interface GameDetailsActionsMenuProps {
  game: StoredGame;
  isOpen: boolean;
  onClearStorageOverride: () => void;
  onSetStorageOverride: () => void;
}

export const GameDetailsActionsMenu = ({
  game,
  isOpen,
  onClearStorageOverride,
  onSetStorageOverride,
}: GameDetailsActionsMenuProps) => {
  if (!isOpen) {
    return null;
  }

  const hasOverride = game.ModStoragePathOverride !== null && game.ModStoragePathOverride.trim() !== '';

  return (
    <div className="game-details-actions-menu" role="menu" aria-label="Game actions">
      <button
        className="game-details-actions-menu-item"
        onClick={onSetStorageOverride}
        role="menuitem"
        type="button"
      >
        <FolderCog className="game-details-actions-menu-icon" aria-hidden="true" />
        Set mod storage override
      </button>

      <button
        className="game-details-actions-menu-item"
        disabled={!hasOverride}
        onClick={onClearStorageOverride}
        role="menuitem"
        type="button"
      >
        <RotateCcw className="game-details-actions-menu-icon" aria-hidden="true" />
        Clear mod storage override
      </button>
    </div>
  );
};
