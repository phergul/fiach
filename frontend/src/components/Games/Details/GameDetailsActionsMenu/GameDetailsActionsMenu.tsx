import { FolderCog, RotateCcw } from 'lucide-react';

import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';

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
    <DropdownMenu
      ariaLabel="Game actions"
      isOpen={isOpen}
      items={[
        {
          icon: FolderCog,
          label: 'Set mod storage override',
          onSelect: onSetStorageOverride,
        },
        {
          disabled: !hasOverride,
          icon: RotateCcw,
          label: 'Clear mod storage override',
          onSelect: onClearStorageOverride,
        },
      ]}
    />
  );
};
