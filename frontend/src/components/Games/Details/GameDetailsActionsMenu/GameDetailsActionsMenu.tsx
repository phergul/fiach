import { FolderCog, RotateCcw, Sparkles } from 'lucide-react';

import type { StoredGame } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';

import './GameDetailsActionsMenu.scss';

interface GameDetailsActionsMenuProps {
  game: StoredGame;
  isOpen: boolean;
  onOpenReShadeInstaller: () => void;
  onClearStorageOverride: () => void;
  onSetStorageOverride: () => void;
  reShadeInstallerActionLabel: string | null;
}

export const GameDetailsActionsMenu = ({
  game,
  isOpen,
  onOpenReShadeInstaller,
  onClearStorageOverride,
  onSetStorageOverride,
  reShadeInstallerActionLabel,
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
        ...(reShadeInstallerActionLabel !== null ? [{
          icon: Sparkles,
          label: reShadeInstallerActionLabel,
          onSelect: onOpenReShadeInstaller,
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
