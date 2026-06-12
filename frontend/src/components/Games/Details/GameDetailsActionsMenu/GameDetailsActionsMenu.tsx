import { Blocks, FolderCog, RotateCcw, Sparkles } from 'lucide-react';

import type { StoredGame } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';

import './GameDetailsActionsMenu.scss';

interface GameDetailsActionsMenuProps {
  game: StoredGame;
  isOpen: boolean;
  onOpenReShadeAddonInstaller: () => void;
  onOpenReShadeInstaller: () => void;
  onClearStorageOverride: () => void;
  onSetStorageOverride: () => void;
  reShadeAddonInstallerActionLabel: string | null;
  reShadeInstallerActionLabel: string | null;
}

export const GameDetailsActionsMenu = ({
  game,
  isOpen,
  onOpenReShadeAddonInstaller,
  onOpenReShadeInstaller,
  onClearStorageOverride,
  onSetStorageOverride,
  reShadeAddonInstallerActionLabel,
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
        ...(reShadeAddonInstallerActionLabel !== null ? [{
          icon: Blocks,
          label: reShadeAddonInstallerActionLabel,
          onSelect: onOpenReShadeAddonInstaller,
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
