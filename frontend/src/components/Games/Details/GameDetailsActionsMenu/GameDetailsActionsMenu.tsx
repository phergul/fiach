import { Blocks, FolderCog, Gauge, RotateCcw, SlidersHorizontal, Sparkles } from 'lucide-react';

import type { StoredGame } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';

import './GameDetailsActionsMenu.scss';

interface GameDetailsActionsMenuProps {
  game: StoredGame;
  isOpen: boolean;
  onOpenOptiScaler: () => void;
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
  onOpenOptiScaler,
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
  const supportsWindowsGraphicsTools =
    reShadeInstallerActionLabel !== null || reShadeAddonInstallerActionLabel !== null;

  return (
    <DropdownMenu
      ariaLabel="Game actions"
      isOpen={isOpen}
      items={[
        {
          children: [
            ...(supportsWindowsGraphicsTools ? [{
              icon: Gauge,
              label: 'OptiScaler',
              onSelect: onOpenOptiScaler,
            }] : []),
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
