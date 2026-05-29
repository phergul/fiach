import { FolderCog, RotateCcw, Sparkles } from 'lucide-react';

import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';

import './GameDetailsActionsMenu.scss';

interface GameDetailsActionsMenuProps {
  game: StoredGame;
  isOpen: boolean;
  onInstallReShade: () => void;
  onClearStorageOverride: () => void;
  onSetStorageOverride: () => void;
  showInstallReShade: boolean;
}

export const GameDetailsActionsMenu = ({
  game,
  isOpen,
  onInstallReShade,
  onClearStorageOverride,
  onSetStorageOverride,
  showInstallReShade,
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
        ...(showInstallReShade ? [{
          icon: Sparkles,
          label: 'Install ReShade',
          onSelect: onInstallReShade,
        }] : []),
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
