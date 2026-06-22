import { FolderCog, Gauge, RotateCcw, SlidersHorizontal, Sparkles } from 'lucide-react';

import type { StoredGame } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';

import './GameDetailsActionsMenu.scss';

interface GameDetailsActionsMenuProps {
  game: StoredGame;
  isOpen: boolean;
  onOpenOptiScaler: () => void;
  onOpenReShade: () => void;
  onClearStorageOverride: () => void;
  onSetStorageOverride: () => void;
}

export const GameDetailsActionsMenu = ({
  game,
  isOpen,
  onOpenOptiScaler,
  onOpenReShade,
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
          children: [
            {
              icon: Gauge,
              label: 'OptiScaler',
              onSelect: onOpenOptiScaler,
            },
            {
              icon: Sparkles,
              label: 'ReShade',
              onSelect: onOpenReShade,
            },
          ],
          icon: SlidersHorizontal,
          label: 'Manage graphics tools',
        },
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
