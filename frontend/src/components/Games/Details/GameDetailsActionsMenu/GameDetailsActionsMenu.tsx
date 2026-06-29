import { FolderCog, FolderOpen, Gauge, RotateCcw, SlidersHorizontal, Sparkles } from 'lucide-react';

import type { StoredGame } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { DropdownMenu, type DropdownMenuItem } from '@components/Common/DropdownMenu/DropdownMenu';
import { useRuntime } from '@hooks';

import './GameDetailsActionsMenu.scss';

interface GameDetailsActionsMenuProps {
  game: StoredGame;
  isOpen: boolean;
  onOpenOptiScaler: () => void;
  onOpenReShade: () => void;
  onClearStorageOverride: () => void;
  onOpenInstallDirectory: () => void;
  onSetStorageOverride: () => void;
}

export const GameDetailsActionsMenu = ({
  game,
  isOpen,
  onOpenOptiScaler,
  onOpenReShade,
  onClearStorageOverride,
  onOpenInstallDirectory,
  onSetStorageOverride,
}: GameDetailsActionsMenuProps) => {
  const { isWindows } = useRuntime();

  if (!isOpen) {
    return null;
  }

  const hasOverride =
    game.ModStoragePathOverride !== null && game.ModStoragePathOverride.trim() !== '';

  const items: DropdownMenuItem[] = [
    {
      icon: FolderOpen,
      label: 'Open install folder',
      onSelect: onOpenInstallDirectory,
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
  ];

  if (isWindows) {
    items.unshift({
      icon: SlidersHorizontal,
      label: 'Manage graphics tools',
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
    });
  }

  return <DropdownMenu ariaLabel="Game actions" isOpen={isOpen} items={items} />;
};
